/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/yaml"
)

const KofRulesClusterNameLabel = "k0rdent.mirantis.com/kof-rules-cluster-name"
const DefaultClusterName = ""
const ReleaseNameAnnotation = "meta.helm.sh/release-name"
const ReleaseNameLabel = "app.kubernetes.io/instance"

// We're going to merge all the rules into the nested map:
// clusterGroupRules[clusterName][groupName][ruleName] = promv1.Rule{}
type Rules map[string]promv1.Rule
type GroupRules map[string]Rules
type ClusterGroupRules map[string]GroupRules

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch
type ConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			_, hasLabel := obj.GetLabels()[KofRulesClusterNameLabel]
			return hasLabel
		})).
		Complete(r)
}

func (r *ConfigMapReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	// If any ConfigMap with the required label exists when kof-operator starts,
	// or such ConfigMap is created or updated later,
	// update the resulting ConfigMap mounted as `/etc/promxy/rules`.
	if err := r.updatePromxyRulesConfigMap(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Update the ConfigMap mounted as `/etc/promxy/rules` with rules based on the
// PrometheusRules, default ConfigMap, and the cluster-specific ConfigMaps.
// See `charts/kof-mothership/templates/promxy/rules.yaml` for more details.
func (r *ConfigMapReconciler) updatePromxyRulesConfigMap(ctx context.Context) error {
	log := log.FromContext(ctx)

	releaseNamespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}

	releaseName, ok := os.LookupEnv("RELEASE_NAME")
	if !ok {
		return fmt.Errorf("required RELEASE_NAME env var is not set")
	}

	clusterGroupRules := ClusterGroupRules{}
	clusterGroupRules[DefaultClusterName] = GroupRules{}

	err := r.mergePrometheusRules(ctx, clusterGroupRules, releaseNamespace, releaseName)
	if err != nil {
		return err
	}

	err = r.mergeConfigMaps(ctx, clusterGroupRules, releaseNamespace)
	if err != nil {
		return err
	}

	files, err := r.convertClusterGroupRulesToFiles(ctx, clusterGroupRules)
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{}
	namespacedName := types.NamespacedName{
		Namespace: releaseNamespace,
		Name:      releaseName + "-promxy-rules",
	}

	if err := r.Get(ctx, namespacedName, configMap); err != nil ||
		configMap.Annotations[ReleaseNameAnnotation] != releaseName {
		log.Error(err, "failed to get ConfigMap installed by the release",
			"configMap", namespacedName,
			ReleaseNameAnnotation, releaseName,
		)
		return err
	}

	if maps.Equal(configMap.Data, files) {
		utils.LogEvent(
			ctx,
			"ConfigMapNoUpdateNeeded",
			"No need to update ConfigMap",
			configMap,
			nil,
			"configMapName", configMap.Name,
		)
		return nil
	}

	configMap.Data = files
	if err := r.Update(ctx, configMap); err != nil {
		utils.LogEvent(
			ctx,
			"ConfigMapUpdateFailed",
			"Failed to update ConfigMap",
			configMap,
			err,
			"configMapName", configMap.Name,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"ConfigMapUpdated",
		"ConfigMap is successfully updated",
		configMap,
		nil,
		"configMapName", configMap.Name,
	)
	return nil
}

func (r *ConfigMapReconciler) mergePrometheusRules(
	ctx context.Context,
	clusterGroupRules ClusterGroupRules,
	releaseNamespace string,
	releaseName string,
) error {
	log := log.FromContext(ctx)

	prometheusRuleList := &promv1.PrometheusRuleList{}
	if err := r.List(
		ctx,
		prometheusRuleList,
		client.InNamespace(releaseNamespace),
		client.MatchingLabels{ReleaseNameLabel: releaseName},
	); err != nil {
		log.Error(
			err, "failed to list PrometheusRules",
			ReleaseNameLabel, releaseName,
		)
		return err
	}

	for _, prometheusRule := range prometheusRuleList.Items {
		for _, ruleGroup := range prometheusRule.Spec.Groups {
			groupName := ruleGroup.Name
			rules, ok := clusterGroupRules[DefaultClusterName][groupName]
			if !ok {
				rules = Rules{}
				clusterGroupRules[DefaultClusterName][groupName] = rules
			}
			for _, rule := range ruleGroup.Rules {
				// Process alerting rules only.
				ruleName := rule.Alert
				if ruleName != "" {
					rules[ruleName] = rule
				}
			}
		}
	}

	return nil
}

func (r *ConfigMapReconciler) mergeConfigMaps(
	ctx context.Context,
	clusterGroupRules ClusterGroupRules,
	releaseNamespace string,
) error {
	log := log.FromContext(ctx)

	configMapList := &corev1.ConfigMapList{}
	if err := r.List(
		ctx,
		configMapList,
		client.InNamespace(releaseNamespace),
		client.HasLabels{KofRulesClusterNameLabel},
	); err != nil {
		log.Error(
			err, "failed to list ConfigMaps",
			"label", KofRulesClusterNameLabel,
		)
		return err
	}
	configMaps := configMapList.Items

	// Move ConfigMaps with DefaultClusterName to the beginning of the list,
	// as we want to merge them first.
	defaultConfigMaps := make([]corev1.ConfigMap, 0, len(configMaps))
	clusterConfigMaps := make([]corev1.ConfigMap, 0, len(configMaps))
	for _, configMap := range configMaps {
		if configMap.Labels[KofRulesClusterNameLabel] == DefaultClusterName {
			defaultConfigMaps = append(defaultConfigMaps, configMap)
		} else {
			clusterConfigMaps = append(clusterConfigMaps, configMap)
		}
	}
	configMaps = append(defaultConfigMaps, clusterConfigMaps...)

	// Merge the rules from ConfigMaps to the clusterGroupRules map.
	for _, configMap := range configMaps {
		clusterName := configMap.Labels[KofRulesClusterNameLabel]
		groupRules, ok := clusterGroupRules[clusterName]
		if !ok {
			groupRules = GroupRules{}
			clusterGroupRules[clusterName] = groupRules
		}
		for groupName, rulesYAML := range configMap.Data {
			rules, ok := groupRules[groupName]
			if !ok {
				rules = Rules{}
				groupRules[groupName] = rules
			}

			newRules := make(map[string]promv1.Rule)
			if err := yaml.Unmarshal([]byte(rulesYAML), &newRules); err != nil {
				log.Error(
					err, "failed to unmarshal rules",
					"cluster", clusterName,
					"group", groupName,
					"rules", rulesYAML,
				)
				return err
			}

			for ruleName, newRule := range newRules {
				newRule.Alert = ruleName

				oldRule, ok := rules[ruleName]
				if ok {
					// No need for deep copy here:
					// default ConfigMap should overwrite the data loaded from PrometheusRules,
					// and cluster-specific ConfigMap will patch its own cluster rules only.
					patchRule(&oldRule, &newRule)
					rules[ruleName] = oldRule
					continue
				}

				if clusterName != DefaultClusterName {
					defaultRules, ok := clusterGroupRules[DefaultClusterName][groupName]
					if ok {
						defaultRule, ok := defaultRules[ruleName]
						if ok {
							defaultRuleCopyPtr := defaultRule.DeepCopy()
							patchRule(defaultRuleCopyPtr, &newRule)
							rules[ruleName] = *defaultRuleCopyPtr
							continue
						}
					}
				}

				rules[ruleName] = newRule
			}
		}
	}

	return nil
}

func patchRule(oldRule *promv1.Rule, newRule *promv1.Rule) {
	if newRule.Expr.String() != "" {
		oldRule.Expr = newRule.Expr
	}
	if newRule.For != nil {
		oldRule.For = newRule.For
	}
	if newRule.KeepFiringFor != nil {
		oldRule.KeepFiringFor = newRule.KeepFiringFor
	}
	if newRule.Labels != nil {
		if oldRule.Labels == nil {
			oldRule.Labels = make(map[string]string, len(newRule.Labels))
		}
		maps.Copy(oldRule.Labels, newRule.Labels)
	}
	if newRule.Annotations != nil {
		if oldRule.Annotations == nil {
			oldRule.Annotations = make(map[string]string, len(newRule.Annotations))
		}
		maps.Copy(oldRule.Annotations, newRule.Annotations)
	}
}

func (r *ConfigMapReconciler) convertClusterGroupRulesToFiles(
	ctx context.Context,
	clusterGroupRules ClusterGroupRules,
) (map[string]string, error) {
	log := log.FromContext(ctx)
	files := map[string]string{}

	for clusterName, groupRules := range clusterGroupRules {
		for groupName, rules := range groupRules {
			fileName := groupName + ".yaml"
			if clusterName != DefaultClusterName {
				fileName = fmt.Sprintf("%s__%s", clusterName, fileName)
			}

			rulesSlice := slices.Collect(maps.Values(rules))
			slices.SortFunc(rulesSlice, func(a, b promv1.Rule) int {
				return strings.Compare(a.Alert, b.Alert)
			})

			for _, rule := range rulesSlice {
				if rule.Labels == nil {
					rule.Labels = make(map[string]string)
				}
				rule.Labels["alertgroup"] = groupName
				// If we find that adding `{cluster="cluster1"}` to `.Values.clusterRulesPatch`
				// and `{cluster!~"^cluster1$|^cluster10$"}` to `.Values.defaultRulesPatch`
				// manually is a problem, we can update `rule.Expr` automatically here with
				// https://github.com/prometheus/prometheus/blob/main/promql/parser/ast.go
			}

			prometheusRuleSpec := promv1.PrometheusRuleSpec{
				Groups: []promv1.RuleGroup{
					{Name: groupName, Rules: rulesSlice},
				},
			}

			yamlBytes, err := yaml.Marshal(prometheusRuleSpec)
			if err != nil {
				log.Error(
					err, "failed to marshal rules",
					"cluster", clusterName,
					"group", groupName,
				)
				return nil, err
			}
			files[fileName] += string(yamlBytes)
		}
	}

	return files, nil
}
