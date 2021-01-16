package datasource

import (
	"context"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/go-chassis/cari/discovery"
	"reflect"
	"regexp"
	"strings"
)

type RuleFilter struct {
	DomainProject string
	ProviderRules []*discovery.ServiceRule
}

func (rf *RuleFilter) FilterAll(ctx context.Context, consumerIDs []string,
	filterFunc func(ctx2 context.Context, rules []*discovery.ServiceRule,
		consumerID string, domainProject string) (bool, error)) (allow []string, deny []string, err error) {

	l := len(consumerIDs)
	if l == 0 || len(rf.ProviderRules) == 0 {
		return consumerIDs, nil, nil
	}

	allowIdx, denyIdx := 0, l
	consumers := make([]string, l)
	for _, consumerID := range consumerIDs {
		ok, err := filterFunc(ctx, rf.ProviderRules, consumerID, rf.DomainProject)
		//ok, err := rf.Filter(ctx, consumerID)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			consumers[allowIdx] = consumerID
			allowIdx++
		} else {
			denyIdx--
			consumers[denyIdx] = consumerID
		}
	}
	return consumers[:allowIdx], consumers[denyIdx:], nil
}

func MatchRules(rulesOfProvider []*discovery.ServiceRule, consumer *discovery.MicroService, tagsOfConsumer map[string]string) *discovery.Error {
	if consumer == nil {
		return discovery.NewError(discovery.ErrInvalidParams, "consumer is nil")
	}

	if len(rulesOfProvider) <= 0 {
		return nil
	}
	if rulesOfProvider[0].RuleType == "WHITE" {
		return patternWhiteList(rulesOfProvider, tagsOfConsumer, consumer)
	}
	return patternBlackList(rulesOfProvider, tagsOfConsumer, consumer)
}

func patternWhiteList(rulesOfProvider []*discovery.ServiceRule, tagsOfConsumer map[string]string, consumer *discovery.MicroService) *discovery.Error {
	v := reflect.Indirect(reflect.ValueOf(consumer))
	consumerID := consumer.ServiceId
	for _, rule := range rulesOfProvider {
		value, err := parsePattern(v, rule, tagsOfConsumer, consumerID)
		if err != nil {
			return err
		}
		if len(value) == 0 {
			continue
		}

		match, _ := regexp.MatchString(rule.Pattern, value)
		if match {
			log.Infof("consumer[%s][%s/%s/%s/%s] match white list, rule.Pattern is %s, value is %s",
				consumerID, consumer.Environment, consumer.AppId, consumer.ServiceName, consumer.Version,
				rule.Pattern, value)
			return nil
		}
	}
	return discovery.NewError(discovery.ErrPermissionDeny, "Not found in white list")
}

func parsePattern(v reflect.Value, rule *discovery.ServiceRule, tagsOfConsumer map[string]string, consumerID string) (string, *discovery.Error) {
	if strings.HasPrefix(rule.Attribute, "tag_") {
		key := rule.Attribute[4:]
		value := tagsOfConsumer[key]
		if len(value) == 0 {
			log.Infof("can not find service[%s] tag[%s]", consumerID, key)
		}
		return value, nil
	}
	key := v.FieldByName(rule.Attribute)
	if !key.IsValid() {
		log.Errorf(nil, "can not find service[%] field[%s], ruleID is %s",
			consumerID, rule.Attribute, rule.RuleId)
		return "", discovery.NewErrorf(discovery.ErrInternal, "Can not find field '%s'", rule.Attribute)
	}
	return key.String(), nil

}

func patternBlackList(rulesOfProvider []*discovery.ServiceRule, tagsOfConsumer map[string]string, consumer *discovery.MicroService) *discovery.Error {
	v := reflect.Indirect(reflect.ValueOf(consumer))
	consumerID := consumer.ServiceId
	for _, rule := range rulesOfProvider {
		var value string
		value, err := parsePattern(v, rule, tagsOfConsumer, consumerID)
		if err != nil {
			return err
		}
		if len(value) == 0 {
			continue
		}

		match, _ := regexp.MatchString(rule.Pattern, value)
		if match {
			log.Warnf("no permission to access, consumer[%s][%s/%s/%s/%s] match black list, rule.Pattern is %s, value is %s",
				consumerID, consumer.Environment, consumer.AppId, consumer.ServiceName, consumer.Version,
				rule.Pattern, value)
			return discovery.NewError(discovery.ErrPermissionDeny, "Found in black list")
		}
	}
	return nil
}
