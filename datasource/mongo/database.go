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
	"time"

	pb "github.com/go-chassis/cari/discovery"
)

const (
	DuplicateKey               = 11000
	AccountName                = "name"
	AccountID                  = "id"
	AccountPassword            = "password"
	AccountRole                = "role"
	AccountTokenExpirationTime = "tokenexpirationtime"
	AccountCurrentPassword     = "currentpassword"
	AccountStatus              = "status"
	RefreshTime                = "refreshtime"
)

const (
	CollectionAccount  = "account"
	CollectionService  = "service"
	CollectionSchema   = "schema"
	CollectionRule     = "rule"
	CollectionInstance = "instance"
	CollectionDep      = "dependency"
)

const (
	DepsQueueUUID     = "0"
	ErrorDuplicateKey = 11000
)

const (
	ColumnDomain         = "domain"
	ColumnProject        = "project"
	ColumnTag            = "tags"
	ColumnSchemaID       = "schemaid"
	ColumnServiceID      = "serviceid"
	ColumnRuleID         = "ruleid"
	ColumnServiceInfo    = "serviceinfo"
	ColumnProperty       = "properties"
	ColumnModTime        = "modtimestamp"
	ColumnEnv            = "environment"
	ColumnAppID          = "appid"
	ColumnServiceName    = "servicename"
	ColumnAlias          = "alias"
	ColumnVersion        = "version"
	ColumnSchemas        = "schemas"
	ColumnAttribute      = "attribute"
	ColumnPattern        = "pattern"
	ColumnDescription    = "description"
	ColumnRuleType       = "ruletype"
	ColumnSchemaInfo     = "schemainfo"
	ColumnSchemaSummary  = "schemasummary"
	ColumnConsumer       = "consumer"
	ColumnProviders      = "providers"
	ColumnDependencyInfo = "dependencyinfo"
	ColumnRuleInfo       = "ruleinfo"
	ColumnInstanceInfo   = "instanceinfo"
	ColumnInstanceID     = "instanceid"
	ColumnConsumerID     = "consumerid"
	ColumnMongoID        = "_id"
	ColumnTenant         = "tenant"
)

type Service struct {
	Domain      string
	Project     string
	Tags        map[string]string
	ServiceInfo *pb.MicroService
}

type Schema struct {
	Domain        string
	Project       string
	ServiceID     string
	SchemaID      string
	SchemaInfo    string
	SchemaSummary string
}

type Rule struct {
	Domain    string
	Project   string
	ServiceID string
	RuleInfo  *pb.ServiceRule
}

type Instance struct {
	Domain       string
	Project      string
	RefreshTime  time.Time
	InstanceInfo *pb.MicroServiceInstance
}

type Dependency struct {
	Domain         string
	Project        string
	ConsumerID     string
	UUID           string
	DependencyInfo *pb.ConsumerDependency
}
