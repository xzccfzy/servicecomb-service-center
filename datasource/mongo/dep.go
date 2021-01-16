/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongo

import (
	"context"
	"fmt"
	"github.com/apache/servicecomb-service-center/datasource"
	"github.com/apache/servicecomb-service-center/datasource/etcd/path"
	"github.com/apache/servicecomb-service-center/datasource/mongo/client"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/go-chassis/cari/discovery"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (ds *DataSource) SearchProviderDependency(ctx context.Context, request *discovery.GetDependenciesRequest) (*discovery.GetProDependenciesResponse, error) {
	domainProject := util.ParseDomainProject(ctx)
	providerServiceID := request.ServiceId
	filter := GeneratorServiceFilter(ctx, providerServiceID)
	provider, err := GetService(ctx, filter)
	if err != nil {
		log.Error("GetProviderDependencies failed, provider is "+providerServiceID, err)
		return nil, err
	}
	if provider == nil {
		log.Error(fmt.Sprintf("GetProviderDependencies failed for provider %s", providerServiceID), err)
		return &discovery.GetProDependenciesResponse{
			Response: discovery.CreateResponse(discovery.ErrServiceNotExists, "Provider does not exist"),
		}, nil
	}

	dr := NewProviderDependencyRelation(ctx, domainProject, provider.ServiceInfo)
	services, err := dr.GetDependencyConsumers(ToDependencyFilterOptions(request)...)
	if err != nil {
		log.Error(fmt.Sprintf("GetProviderDependencies failed, provider is %s/%s/%s/%s",
			provider.ServiceInfo.Environment, provider.ServiceInfo.AppId, provider.ServiceInfo.ServiceName, provider.ServiceInfo.Version), err)
		return &discovery.GetProDependenciesResponse{
			Response: discovery.CreateResponse(discovery.ErrInternal, err.Error()),
		}, err
	}

	return &discovery.GetProDependenciesResponse{
		Response:  discovery.CreateResponse(discovery.ResponseSuccess, "Get all consumers successful."),
		Consumers: services,
	}, nil
}

func (ds *DataSource) SearchConsumerDependency(ctx context.Context, request *discovery.GetDependenciesRequest) (*discovery.GetConDependenciesResponse, error) {
	domainProject := util.ParseDomainProject(ctx)
	consumerID := request.ServiceId

	filter := GeneratorServiceFilter(ctx, consumerID)
	consumer, err := GetService(ctx, filter)
	if err != nil {
		log.Error(fmt.Sprintf("GetConsumerDependencies failed, consumer is %s", consumerID), err)
		return nil, err
	}
	if consumer == nil {
		log.Error(fmt.Sprintf("GetConsumerDependencies failed for consumer %s does not exist", consumerID), err)
		return &discovery.GetConDependenciesResponse{
			Response: discovery.CreateResponse(discovery.ErrServiceNotExists, "Consumer does not exist"),
		}, nil
	}

	dr := NewConsumerDependencyRelation(ctx, domainProject, consumer.ServiceInfo)
	services, err := dr.GetDependencyProviders(ToDependencyFilterOptions(request)...)
	if err != nil {
		log.Error(fmt.Sprintf("GetConsumerDependencies failed, consumer is %s/%s/%s/%s",
			consumer.ServiceInfo.Environment, consumer.ServiceInfo.AppId, consumer.ServiceInfo.ServiceName, consumer.ServiceInfo.Version), err)
		return &discovery.GetConDependenciesResponse{
			Response: discovery.CreateResponse(discovery.ErrInternal, err.Error()),
		}, err
	}

	return &discovery.GetConDependenciesResponse{
		Response:  discovery.CreateResponse(discovery.ResponseSuccess, "Get all providers successfully."),
		Providers: services,
	}, nil
}

func (ds *DataSource) AddOrUpdateDependencies(ctx context.Context, dependencyInfos []*discovery.ConsumerDependency, override bool) (*discovery.Response, error) {
	domainProject := util.ParseDomainProject(ctx)
	for _, dependencyInfo := range dependencyInfos {
		consumerFlag := util.StringJoin([]string{
			dependencyInfo.Consumer.Environment,
			dependencyInfo.Consumer.AppId,
			dependencyInfo.Consumer.ServiceName,
			dependencyInfo.Consumer.Version}, "/")
		consumerInfo := discovery.DependenciesToKeys([]*discovery.MicroServiceKey{dependencyInfo.Consumer}, domainProject)[0]
		providersInfo := discovery.DependenciesToKeys(dependencyInfo.Providers, domainProject)

		rsp := datasource.ParamsChecker(consumerInfo, providersInfo)
		if rsp != nil {
			log.Error(fmt.Sprintf("put request into dependency queue failed, override: %t consumer is %s %s",
				override, consumerFlag, rsp.Response.GetMessage()), nil)
			return rsp.Response, nil
		}

		consumerID, err := GetServiceID(ctx, consumerInfo)
		if err != nil {
			log.Error(fmt.Sprintf("put request into dependency queue failed, override: %t, get consumer %s id failed",
				override, consumerFlag), err)
			return discovery.CreateResponse(discovery.ErrInternal, err.Error()), err
		}
		if len(consumerID) == 0 {
			log.Error(fmt.Sprintf("put request into dependency queue failed, override: %t consumer %s does not exist",
				override, consumerFlag), err)
			return discovery.CreateResponse(discovery.ErrServiceNotExists, fmt.Sprintf("Consumer %s does not exist.", consumerFlag)), nil
		}

		dependencyInfo.Override = override
		id := DepsQueueUUID
		if !override {
			id = util.GenerateUUID()

			domain := util.ParseDomain(ctx)
			project := util.ParseProject(ctx)
			data := &ConsumerDep{
				Domain:          domain,
				Project:         project,
				ConsumerID:      consumerID,
				UUID:            id,
				ConsumerDepInfo: dependencyInfo,
			}
			insertRes, err := client.GetMongoClient().Insert(ctx, CollectionDep, data)
			if err != nil {
				log.Error("failed to insert dep to mongodb", err)
				return discovery.CreateResponse(discovery.ErrInternal, err.Error()), err
			}
			log.Info(fmt.Sprintf("insert dep to mongodb success %s", insertRes.InsertedID))
		}
		err = syncDependencyRule(ctx, domainProject, dependencyInfo)
		if err != nil {
			return nil, err
		}
	}
	return discovery.CreateResponse(discovery.ResponseSuccess, "Create dependency successfully."), nil
}

func (ds *DataSource) DeleteDependency() {
	panic("implement me")
}

func syncDependencyRule(ctx context.Context, domainProject string, r *discovery.ConsumerDependency) error {

	consumerInfo := discovery.DependenciesToKeys([]*discovery.MicroServiceKey{r.Consumer}, domainProject)[0]
	providersInfo := discovery.DependenciesToKeys(r.Providers, domainProject)

	var dep datasource.Dependency
	//var err error
	dep.DomainProject = domainProject
	dep.Consumer = consumerInfo
	dep.ProvidersRule = providersInfo
	// add mongo get dep here

	oldProviderRules, err := GetOldProviderRules(&dep)
	if err != nil {
		return err
	}

	if r.Override {
		datasource.ParseOverrideRules(ctx, &dep, oldProviderRules)
	} else {
		datasource.ParseAddOrUpdateRules(ctx, &dep, oldProviderRules)
	}
	return updateDeps(domainProject, &dep)
}

func GetOldProviderRules(dep *datasource.Dependency) (*discovery.MicroServiceDependency, error) {
	microServiceDependency := &discovery.MicroServiceDependency{
		Dependency: []*discovery.MicroServiceKey{},
	}
	filter := GenerateConsumerDependencyRuleKey(dep.DomainProject, dep.Consumer)
	findRes, err := client.GetMongoClient().FindOne(context.TODO(), CollectionDep, filter)
	if err != nil {
		log.Error(fmt.Sprintf("get dependency rule [%v] failed", filter), err)
		return nil, err
	}

	if findRes.Err() != nil {
		return microServiceDependency, nil
	}

	var depRule *DependencyRule
	err = findRes.Decode(&depRule)
	if err != nil {
		return nil, err
	}
	return depRule.DepInfo, nil
}

func updateDeps(domainProject string, dep *datasource.Dependency) error {
	var upsert = true
	for _, r := range dep.DeleteDependencyRuleList {
		filter := GenerateProviderDependencyRuleKey(domainProject, r)
		_, err := client.GetMongoClient().Update(context.TODO(), CollectionDep, filter, bson.M{"$pull": bson.M{StringBuilder([]string{ColumnDepInfo, ColumnDependency}): dep.Consumer}})
		if err != nil {
			return err
		}
		if r.ServiceName == "*" {
			break
		}
	}
	for _, r := range dep.CreateDependencyRuleList {
		filter := GenerateProviderDependencyRuleKey(domainProject, r)
		data := bson.M{
			"$addToSet": bson.M{StringBuilder([]string{ColumnDepInfo, ColumnDependency}): dep.Consumer},
		}
		_, err := client.GetMongoClient().Update(context.TODO(), CollectionDep, filter, data, &options.UpdateOptions{Upsert: &upsert})
		if err != nil {
			return err
		}
		if r.ServiceName == "*" {
			break
		}
	}
	filter := GenerateConsumerDependencyRuleKey(domainProject, dep.Consumer)
	if len(dep.ProvidersRule) == 0 {
		_, err := client.GetMongoClient().Delete(context.TODO(), CollectionDep, filter)
		if err != nil {
			return err
		}
	} else {
		updateData := bson.M{
			"$set": bson.M{StringBuilder([]string{ColumnDepInfo, ColumnDependency}): dep.ProvidersRule},
		}
		_, err := client.GetMongoClient().Update(context.TODO(), CollectionDep, filter, updateData, &options.UpdateOptions{Upsert: &upsert})
		if err != nil {
			return err
		}
	}

	err := CleanUpDepRules(context.TODO(), domainProject)
	if err != nil {
		return err
	}

	return nil
}

func CleanUpDepRules(ctx context.Context, domainProject string) error {
	if len(domainProject) == 0 {
		return ErrInvalidDomainProject
	}

	cache := make(map[*DelDepCacheKey]bool)
	err := removeProviderRuleOfConsumer(ctx, domainProject, cache)

	if err != nil {
		return err
	}

	return removeProviderRuleKeys(ctx, domainProject, cache)
}

func removeProviderRuleOfConsumer(ctx context.Context, domainProject string, cache map[*DelDepCacheKey]bool) error {
	filter := GenerateConsumerDependencyRuleKey(domainProject, nil)
	depRules, err := GetDepRules(ctx, filter)
	if err != nil {
		return err
	}
	for _, depRule := range depRules {
		err := removeConsumerDeps(ctx, depRule, cache)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeProviderRuleKeys(ctx context.Context, domainProject string, cache map[*DelDepCacheKey]bool) error {
	filter := GenerateProviderDependencyRuleKey(domainProject, nil)
	depRules, err := GetDepRules(ctx, filter)
	if err != nil {
		return err
	}
	for _, depRule := range depRules {
		err := removeProviderDeps(ctx, depRule, cache)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetDepRules(ctx context.Context, filter bson.M) ([]*DependencyRule, error) {
	findRes, err := client.GetMongoClient().Find(ctx, CollectionDep, filter)
	if err != nil {
		return nil, err
	}

	var depRules []*DependencyRule
	for findRes.Next(ctx) {
		var depRule *DependencyRule
		err := findRes.Decode(&depRule)
		if err != nil {
			return nil, err
		}
		depRules = append(depRules, depRule)
	}
	return depRules, nil
}

func removeProviderDeps(ctx context.Context, depRule *DependencyRule, cache map[*DelDepCacheKey]bool) (err error) {
	id := &DelDepCacheKey{
		key:  depRule.ServiceKey,
		Type: path.DepsConsumer,
	}
	exist, ok := cache[id]
	if !ok {
		if depRule.ServiceKey.ServiceName == "*" {
			return nil
		}

		_, exist, err = FindServiceIds(ctx, depRule.ServiceKey.Version, depRule.ServiceKey)
		if err != nil {
			return err
		}

		cache[id] = exist
	}

	filter := bson.M{
		ColumnServiceKey: depRule.ServiceKey,
	}
	if !exist {
		_, err = client.GetMongoClient().DocDelete(ctx, CollectionDep, filter)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeConsumerDeps(ctx context.Context, depRule *DependencyRule, cache map[*DelDepCacheKey]bool) (err error) {
	var left []*discovery.MicroServiceKey
	for _, key := range depRule.DepInfo.Dependency {
		if key.ServiceName == "*" {
			left = append(left, key)
			continue
		}

		id := &DelDepCacheKey{
			key:  key,
			Type: path.DepsProvider,
		}
		exist, ok := cache[id]
		if !ok {
			_, exist, err = FindServiceIds(ctx, key.Version, key)
			if err != nil {
				return err
			}
			cache[id] = exist
		}

		if exist {
			left = append(left, key)
		}
	}
	if len(depRule.DepInfo.Dependency) == len(left) {
		return nil
	}

	filter := bson.M{
		ColumnServiceKey: depRule.ServiceKey,
	}
	if len(left) == 0 {
		_, err = client.GetMongoClient().DocDelete(ctx, CollectionDep, filter)
	} else {
		updateData := bson.M{
			"$set": bson.M{StringBuilder([]string{ColumnDepInfo, ColumnDependency}): left},
		}
		_, err = client.GetMongoClient().Update(ctx, CollectionDep, filter, updateData)
	}
	if err != nil {
		return err
	}
	return nil
}

func GetServiceID(ctx context.Context, key *discovery.MicroServiceKey) (serviceID string, err error) {
	domain := util.ParseDomain(ctx)
	project := util.ParseProject(ctx)
	filter := bson.M{
		ColumnDomain:  domain,
		ColumnProject: project,
		StringBuilder([]string{ColumnServiceInfo, ColumnEnv}):         key.Environment,
		StringBuilder([]string{ColumnServiceInfo, ColumnAppID}):       key.AppId,
		StringBuilder([]string{ColumnServiceInfo, ColumnServiceName}): key.ServiceName,
		StringBuilder([]string{ColumnServiceInfo, ColumnVersion}):     key.Version}

	findRes, err := client.GetMongoClient().FindOne(ctx, CollectionService, filter)
	if err != nil {
		return
	}
	if findRes.Err() != nil {
		return
	}
	var service *Service
	err = findRes.Decode(&service)
	if err != nil {
		return
	}
	if service == nil {
		return
	}
	return service.ServiceInfo.ServiceId, nil
}

func TransferToMicroServiceDependency(ctx context.Context, filter bson.M) (*discovery.MicroServiceDependency, error) {
	microServiceDependency := &discovery.MicroServiceDependency{
		Dependency: []*discovery.MicroServiceKey{},
	}
	findRes, err := client.GetMongoClient().FindOne(context.TODO(), CollectionDep, filter)
	if err != nil {
		return nil, err
	}
	if findRes.Err() == nil {
		var depRule *DependencyRule
		err := findRes.Decode(&depRule)
		if err != nil {
			return nil, err
		}
		microServiceDependency.Dependency = append(microServiceDependency.Dependency, depRule.DepInfo.Dependency...)
		return microServiceDependency, nil
	}
	return microServiceDependency, nil
}

func GetConsumerDepInfo(ctx context.Context, filter bson.M) ([]*discovery.ConsumerDependency, error) {
	var ConsumerDeps []*discovery.ConsumerDependency

	findRes, err := client.GetMongoClient().Find(context.TODO(), CollectionDep, filter)
	if err != nil {
		return nil, err
	}

	for findRes.Next(ctx) {
		var dep *ConsumerDep
		err = findRes.Decode(&dep)
		if err != nil {
			return nil, err
		}
		ConsumerDeps = append(ConsumerDeps, dep.ConsumerDepInfo)
	}
	return ConsumerDeps, nil

}
