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

const KofAlertRulesClusterNameLabel = "k0rdent.mirantis.com/kof-alert-rules-cluster-name"
const KofRecordRulesClusterNameLabel = "k0rdent.mirantis.com/kof-record-rules-cluster-name"
const KofRecordVMRulesClusterNameLabel = "k0rdent.mirantis.com/kof-record-vmrules-cluster-name"
const DefaultClusterName = ""
const ReleaseNameAnnotation = "meta.helm.sh/release-name"
const ReleaseNameLabel = "app.kubernetes.io/instance"

type AlertRules map[string]promv1.Rule
type RecordRules []promv1.Rule

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch
type ConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Make controller react to `ConfigMaps` having one of expected labels only.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			labels := obj.GetLabels()
			_, isAlert := labels[KofAlertRulesClusterNameLabel]
			_, isRecord := labels[KofRecordRulesClusterNameLabel]
			_, isVMRule := labels[KofRecordVMRulesClusterNameLabel]
			return isAlert || isRecord || isVMRule
		})).
		Complete(r)
}

// When a ConfigMap with one of expected labels is created, updated or deleted,
// update the resulting ConfigMaps.
func (r *ConfigMapReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	if err := r.updateResultingConfigMaps(ctx); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// Update `kof-mothership-promxy-rules` ConfigMap with alert rules.
// Update each `kof-record-vmrules-$regional_cluster_name` ConfigMap with record rules.
// See `charts/kof-mothership/templates/promxy/rules.yaml`
// and `charts/kof-mothership/templates/victoria/record-rules.yaml` for more details.
func (r *ConfigMapReconciler) updateResultingConfigMaps(ctx context.Context) error {
	releaseNamespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}

	releaseName, ok := os.LookupEnv("RELEASE_NAME")
	if !ok {
		return fmt.Errorf("required RELEASE_NAME env var is not set")
	}

	// We're going to merge all alert rules into the nested map:
	// clusterGroupAlertRules[clusterName][groupName][ruleName] = promv1.Rule{}
	clusterGroupAlertRules := map[string]map[string]AlertRules{}
	clusterGroupAlertRules[DefaultClusterName] = map[string]AlertRules{}

	// We're going to merge all record rules into the nested map:
	// clusterGroupRecordRules[clusterName][groupName] = []promv1.Rule{}
	clusterGroupRecordRules := map[string]map[string]RecordRules{}
	clusterGroupRecordRules[DefaultClusterName] = map[string]RecordRules{}

	// Get the input `ConfigMaps`.
	alertConfigMaps, err := r.getConfigMaps(
		ctx, releaseNamespace, KofAlertRulesClusterNameLabel,
	)
	if err != nil {
		return err
	}
	recordConfigMaps, err := r.getConfigMaps(
		ctx, releaseNamespace, KofRecordRulesClusterNameLabel,
	)
	if err != nil {
		return err
	}

	// Get the output `ConfigMaps`.
	// TODO: Revisit namespaces after multi-tenancy is implemented.
	// These `ConfigMaps` are owned by `ClusterDeployments` that may be in a different namespace.
	vmRuleConfigMaps, err := r.getConfigMaps(ctx, "", KofRecordVMRulesClusterNameLabel)
	if err != nil {
		return err
	}

	// Merge alert and record `PrometheusRules` into the nested maps.
	err = r.mergePrometheusRules(ctx,
		releaseNamespace, releaseName,
		clusterGroupAlertRules, clusterGroupRecordRules,
	)
	if err != nil {
		return err
	}

	// Merge alert and record `ConfigMaps` into the nested maps.
	err = mergeAlertConfigMaps(ctx, alertConfigMaps, clusterGroupAlertRules)
	if err != nil {
		return err
	}
	err = mergeRecordConfigMaps(ctx, recordConfigMaps, clusterGroupRecordRules)
	if err != nil {
		return err
	}

	// Update `kof-mothership-promxy-rules` ConfigMap with the files from the nested map.
	alertFiles, err := getAlertFiles(ctx, clusterGroupAlertRules)
	if err != nil {
		return err
	}
	err = r.updateConfigMap(
		ctx, nil, releaseNamespace, releaseName+"-promxy-rules", alertFiles,
	)
	if err != nil {
		return err
	}

	// Update each `kof-record-vmrules-$regional_cluster_name` ConfigMap from the nested map.
	for _, vmRuleConfigMap := range vmRuleConfigMaps {
		err := r.updateRecordVMRulesConfigMap(ctx, clusterGroupRecordRules, &vmRuleConfigMap)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get `ConfigMaps` with the given `label` and optional `namespace`.
// Default `ConfigMaps` are moved to the beginning of the list,
// as we want to merge them first.
func (r *ConfigMapReconciler) getConfigMaps(
	ctx context.Context,
	namespace string,
	label string,
) ([]corev1.ConfigMap, error) {
	log := log.FromContext(ctx)

	options := []client.ListOption{client.HasLabels{label}}
	if namespace != "" {
		options = append(options, client.InNamespace(namespace))
	}

	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, options...); err != nil {
		log.Error(
			err, "failed to list ConfigMaps",
			"label", label,
			"namespace", namespace,
		)
		return nil, err
	}
	configMaps := configMapList.Items

	defaultConfigMaps := make([]corev1.ConfigMap, 0, len(configMaps))
	clusterConfigMaps := make([]corev1.ConfigMap, 0, len(configMaps))
	for _, configMap := range configMaps {
		if configMap.Labels[label] == DefaultClusterName {
			defaultConfigMaps = append(defaultConfigMaps, configMap)
		} else {
			clusterConfigMaps = append(clusterConfigMaps, configMap)
		}
	}
	return append(defaultConfigMaps, clusterConfigMaps...), nil
}

// Merge `PrometheusRules` to the nested maps.
func (r *ConfigMapReconciler) mergePrometheusRules(
	ctx context.Context,
	namespace string,
	releaseName string,
	clusterGroupAlertRules map[string]map[string]AlertRules,
	clusterGroupRecordRules map[string]map[string]RecordRules,
) error {
	log := log.FromContext(ctx)

	prometheusRuleList := &promv1.PrometheusRuleList{}
	if err := r.List(
		ctx,
		prometheusRuleList,
		client.InNamespace(namespace),
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

			alertRules, ok := clusterGroupAlertRules[DefaultClusterName][groupName]
			if !ok {
				alertRules = AlertRules{}
			}

			recordRules, ok := clusterGroupRecordRules[DefaultClusterName][groupName]
			if !ok {
				recordRules = RecordRules{}
			}

			for _, rule := range ruleGroup.Rules {
				if rule.Alert != "" {
					// To avoid `field keep_firing_for not found in type rulefmt.RuleNode` in promxy:
					rule.KeepFiringFor = nil
					alertRules[rule.Alert] = rule
				}
				if rule.Record != "" {
					recordRules = append(recordRules, rule)
				}
			}

			if len(alertRules) > 0 {
				clusterGroupAlertRules[DefaultClusterName][groupName] = alertRules
			}

			if len(recordRules) > 0 {
				clusterGroupRecordRules[DefaultClusterName][groupName] = recordRules
			}
		}
	}

	return nil
}

// Merge the alert rules from `ConfigMaps` to the nested map.
func mergeAlertConfigMaps(
	ctx context.Context,
	alertConfigMaps []corev1.ConfigMap,
	clusterGroupAlertRules map[string]map[string]AlertRules,
) error {
	for _, configMap := range alertConfigMaps {
		clusterName := configMap.Labels[KofAlertRulesClusterNameLabel]
		groupAlertRules, ok := clusterGroupAlertRules[clusterName]
		if !ok {
			groupAlertRules = map[string]AlertRules{}
			clusterGroupAlertRules[clusterName] = groupAlertRules
		}
		for groupName, alertRulesYAML := range configMap.Data {
			alertRules, ok := groupAlertRules[groupName]
			if !ok {
				alertRules = AlertRules{}
				groupAlertRules[groupName] = alertRules
			}

			newAlertRules, err := unmarshalRules[AlertRules](
				ctx, &configMap, clusterName, groupName, alertRulesYAML,
			)
			if err != nil {
				return err
			}

			for ruleName, newRule := range newAlertRules {
				newRule.Alert = ruleName

				oldRule, ok := alertRules[ruleName]
				if ok {
					// No need for deep copy here:
					// default `ConfigMap should overwrite the data loaded from PrometheusRules,
					// and cluster-specific ConfigMap will patch its own cluster rules only.
					patchRule(&oldRule, &newRule)
					alertRules[ruleName] = oldRule
					continue
				}

				if clusterName != DefaultClusterName {
					defaultRules, ok := clusterGroupAlertRules[DefaultClusterName][groupName]
					if ok {
						defaultRule, ok := defaultRules[ruleName]
						if ok {
							defaultRuleCopyPtr := defaultRule.DeepCopy()
							patchRule(defaultRuleCopyPtr, &newRule)
							alertRules[ruleName] = *defaultRuleCopyPtr
							continue
						}
					}
				}

				alertRules[ruleName] = newRule
			}
		}
	}

	return nil
}

// Merge the record rules from `ConfigMaps` to the nested map.
func mergeRecordConfigMaps(
	ctx context.Context,
	recordConfigMaps []corev1.ConfigMap,
	clusterGroupRecordRules map[string]map[string]RecordRules,
) error {
	for _, configMap := range recordConfigMaps {
		clusterName := configMap.Labels[KofRecordRulesClusterNameLabel]
		groupRecordRules, ok := clusterGroupRecordRules[clusterName]
		if !ok {
			groupRecordRules = map[string]RecordRules{}
			clusterGroupRecordRules[clusterName] = groupRecordRules
		}
		for groupName, recordRulesYAML := range configMap.Data {
			recordRules, err := unmarshalRules[RecordRules](
				ctx, &configMap, clusterName, groupName, recordRulesYAML,
			)
			if err != nil {
				return err
			}
			groupRecordRules[groupName] = recordRules
		}
	}
	return nil
}

// Unmarshal `rulesYAML` into the `AlertRules` or `RecordRules`.
func unmarshalRules[T AlertRules | RecordRules](
	ctx context.Context,
	configMap *corev1.ConfigMap,
	clusterName string,
	groupName string,
	rulesYAML string,
) (T, error) {
	var rules T
	if err := yaml.Unmarshal([]byte(rulesYAML), &rules); err != nil {
		utils.LogEvent(
			ctx,
			"RulesUnmarshalFailed",
			"Failed to unmarshal rules",
			configMap,
			err,
			"cluster", clusterName,
			"group", groupName,
			"rules", rulesYAML,
		)
		return rules, err
	}
	return rules, nil
}

// Patch `oldRule` with `newRule`.
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

// Get a map with YAML files from the nested map.
func getAlertFiles(
	ctx context.Context,
	clusterGroupAlertRules map[string]map[string]AlertRules,
) (map[string]string, error) {
	log := log.FromContext(ctx)
	files := map[string]string{}

	for clusterName, groupRules := range clusterGroupAlertRules {
		for groupName, rules := range groupRules {
			fileName := groupName + ".yaml"
			if clusterName != DefaultClusterName {
				fileName = fmt.Sprintf("__%s__%s", clusterName, fileName)
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
					err, "failed to marshal alert rules",
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

// Update `ConfigMap` generated by `kof-operator` with the given `data`.
// Either `configMap` or `namespace` and `name` should be provided.
func (r *ConfigMapReconciler) updateConfigMap(
	ctx context.Context,
	configMap *corev1.ConfigMap,
	namespace string,
	name string,
	data map[string]string,
) error {
	log := log.FromContext(ctx)

	if configMap == nil {
		configMap = &corev1.ConfigMap{}
		namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
		if err := r.Get(ctx, namespacedName, configMap); err != nil {
			log.Error(err, "failed to get ConfigMap",
				"configMap", namespacedName,
			)
			return err
		}
	}

	namespacedName := types.NamespacedName{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}

	if maps.Equal(configMap.Data, data) {
		log.Info("No need to update ConfigMap",
			"configMap", namespacedName,
		)
		return nil
	}

	if configMap.Labels[utils.KofGeneratedLabel] != "true" {
		log.Info("ConfigMap is not generated by kof-operator, skipping update",
			"configMap", namespacedName,
			"label", utils.KofGeneratedLabel,
		)
		return nil
	}

	configMap.Data = data
	if err := r.Update(ctx, configMap); err != nil {
		utils.LogEvent(
			ctx,
			"ConfigMapUpdateFailed",
			"Failed to update ConfigMap",
			configMap,
			err,
			"configMap", namespacedName,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"ConfigMapUpdated",
		"ConfigMap is successfully updated",
		configMap,
		nil,
		"configMap", namespacedName,
	)
	return nil
}

// Update `kof-record-vmrules-$regional_cluster_name` ConfigMap from the nested map.
func (r *ConfigMapReconciler) updateRecordVMRulesConfigMap(
	ctx context.Context,
	clusterGroupRecordRules map[string]map[string]RecordRules,
	resultConfigMap *corev1.ConfigMap,
) error {
	log := log.FromContext(ctx)
	groups := map[string]RecordRules{}

	for groupName, recordRules := range clusterGroupRecordRules[DefaultClusterName] {
		groups[groupName] = recordRules
	}

	clusterName := resultConfigMap.Labels[KofRecordVMRulesClusterNameLabel]
	groupRecordRules, ok := clusterGroupRecordRules[clusterName]
	if ok {
		for groupName, recordRules := range groupRecordRules {
			// Use cluster-specific record rules instead of default ones.
			groups[groupName] = recordRules
		}
	}

	// Don't wrap `vmrules` in `victoriametrics` top-level key,
	// because Sveltos concatenates (not merges) `values` and `valuesFrom`:
	//   victoriametrics:
	//     vmauth:
	//       enabled: false
	//   victoriametrics:  # AGAIN!
	//     vmrules:
	//       groups:
	// This results in only the last occurrence of `victoriametrics` being applied.
	values := map[string]interface{}{
		"vmrules": map[string]interface{}{
			"groups": groups,
		},
	}

	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		log.Error(
			err, "failed to marshal record rules",
			"cluster", clusterName,
		)
		return err
	}

	data := map[string]string{"values": string(valuesYAML)}
	return r.updateConfigMap(ctx, resultConfigMap, "", "", data)
}
