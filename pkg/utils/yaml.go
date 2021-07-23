package utils

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/manifest"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

// GetManifestFromMultiDoc returns an individual object string from a multi-doc yaml file
// after searching by kind. Note: the first object in multiManifest matching kind is returned.
func GetManifestFromMultiDoc(multiDoc, kind string) (string, error) {
	objs, err := manifest.ParseObjects(context.TODO(), multiDoc)
	if err != nil {
		return "", err
	}
	for _, obj := range objs.Items {
		if obj.UnstructuredObject().GetKind() == kind {
			objYaml, err := yaml.Marshal(obj.UnstructuredObject())
			if err != nil {
				return "", err
			}
			return string(objYaml), nil
		}
	}
	return "", fmt.Errorf("no object of kind: %s found in multi doc manifest", kind)
}

// GetAllManifestsOfKindFromMultiDoc returns a slice of strings from a multi-doc yaml file
// after searching by kind. Each string represents a sinlge manifest of 'kind'.
func GetAllManifestsOfKindFromMultiDoc(multiDoc, kind string) ([]string, error) {
	objs, err := manifest.ParseObjects(context.TODO(), multiDoc)
	if err != nil {
		return nil, err
	}
	objsOfKind := make([]string, 0)
	for _, obj := range objs.Items {
		if obj.UnstructuredObject().GetKind() == kind {
			objYaml, err := yaml.Marshal(obj.UnstructuredObject())
			if err != nil {
				return nil, err
			}
			objsOfKind = append(objsOfKind, string(objYaml))
		}
	}
	return objsOfKind, nil
}

// SetFieldInManifest sets valueName equal to value at path in manifest defined by fields.
// See TestSetFieldInManifest for examples.
func SetFieldInManifest(manifest, value, valueName string, fields ...string) (string, error) {
	obj, err := kyaml.Parse(manifest)
	if err != nil {
		return "", err
	}

	parsedVal, err := kyaml.Parse(value)
	if err != nil {
		return "", err
	}

	_, err = obj.Pipe(kyaml.LookupCreate(kyaml.MappingNode, fields...), kyaml.SetField(valueName, parsedVal))
	if err != nil {
		return "", err
	}
	return obj.MustString(), nil

}

// GetFieldInManifest returns the string value at path in manifest defined by fields.
// See TestGetFieldInManifest for examples.
func GetFieldInManifest(manifest string, fields ...string) (string, error) {
	obj, err := kyaml.Parse(manifest)
	if err != nil {
		return "", err
	}

	val, err := obj.Pipe(kyaml.Lookup(fields...))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(val.MustString()), nil
}

// KustomizePatch is useed to pass a new patch to a kustomization file, see AddPatchesToKustomize
type KustomizePatch struct {
	Op    string
	Path  string
	Value string
}

// AddPatchesToKustomize adds any number of patches (via []KustomizePatch{}) to kustomizationFile string,
// returning the updated kustomization file as a string.

// Example
//*******************************************************
// Input kustomization file:
//*******************************************************
// apiVersion: kustomize.config.k8s.io/v1beta1
// kind: Kustomization
//
// resources:
// - storageos-cluster.yaml
//******************************************************
// Other inputs:
// targetKind: "StorageOSCluster"
// targetName: "storageoscluster-sample"
// patches: []KustomizePatch{
//	{
//		Op: "replace",
//		Path: "/spec/kvBackend/address",
//		Value: 	"storageos.storageos-etcd:2379",
//	},
// }
//*******************************************************
// Results in the following output kustomization file:
//*******************************************************
// apiVersion: kustomize.config.k8s.io/v1beta1
// kind: Kustomization
//
// resources:
// - storageos-cluster.yaml
//
// patches:
// - target:
//     kind: StorageOSCluster
//     name: storageoscluster-sample
//   patch: |
//     - op: replace
//       path: /spec/kvBackend/address
//       value: storageos.storageos-etcd:2379
//*******************************************************
func AddPatchesToKustomize(kustomizationFile, targetKind, targetName string, patches []KustomizePatch) (string, error) {
	obj, err := kyaml.Parse(string(kustomizationFile))
	if err != nil {
		return "", err
	}

	patchStrings := make([]string, 0)
	for _, patch := range patches {
		patchString := fmt.Sprintf("%s%s%s%s%s%s", `
    - op: `, patch.Op, `
      path: `, patch.Path, `
      value: `, patch.Value)
		patchStrings = append(patchStrings, patchString)

	}

	allPatchesStr := strings.Join(patchStrings, "")

	targetString := fmt.Sprintf("%s%s%s%s%s", `
- target:
    kind: `, targetKind, `
    name: `, targetName, `
  patch: |`)

	patch, err := kyaml.Parse(strings.Join([]string{targetString, allPatchesStr}, ""))

	_, err = obj.Pipe(
		kyaml.LookupCreate(kyaml.SequenceNode, "patches"),
		kyaml.Append(patch.YNode().Content...))
	if err != nil {
		return "", err
	}

	return obj.MustString(), nil
}

// GenericPatchesForSupportBundle creates and returns []KustomizePatch for a kustomiziation file to be applied to the
// SupportBundle.
//
// Inputs:
// * spec: string of the SupportBundle manifest
// * instruction: "collectors" or "analyzers"
// * value: string of Value for patch
// * fields: path of fields (after instruction) to value to be changed in SupportBundle eg {"namespace"}
// * lookUpValue: value to compare at path skipByFields eg "storageos-operator-logs". If lookup value is left empty,
// any instruction with skipByFields path is skipped. This value is only to specify a single instruction for ignoring.
// * pathsToSkip: (optional) include paths of fields for an instructions to be ignored (ie no patch applied even if it
// matches 'fields' path above. Eg {{"logs"},{"run"}}
//
// This function is useful in cases where it is desired to set a field such as namespace in a SupportBundle for most
// (but not all instructions). The appropriate patches are created and can then be added to the applicable kustomization.
func GenericPatchesForSupportBundle(spec, instruction, value string, fields []string, skipLookUpValue string, pathsToSkip [][]string) ([]KustomizePatch, error) {
	instructionTypes, err := getSupportBundleInstructionTypes(instruction)
	if err != nil {
		return nil, err
	}

	obj, err := kyaml.Parse(spec)
	if err != nil {
		return nil, err
	}
	instructionObj, err := obj.Pipe(kyaml.Lookup(
		"spec",
		instruction,
	))
	if err != nil {
		return nil, err
	}
	instructionPatches := make([]KustomizePatch, 0)
	elements, _ := instructionObj.Elements()
	for count, element := range elements {
		skipElement, err := skipElement(element, pathsToSkip, skipLookUpValue)
		if err != nil {
			return nil, err
		}
		if skipElement {
			continue
		}
		for _, instructionType := range instructionTypes {

			instructionNode, err := element.Pipe(kyaml.Lookup(instructionType))
			if err != nil {
				return nil, err
			}
			if instructionNode == nil {
				continue
			}

			fieldNode, err := instructionNode.Pipe(kyaml.Lookup(fields...))
			if err != nil {

				return nil, err
			}
			if fieldNode == nil {
				break
			}
			path := filepath.Join("/spec", instruction, strconv.Itoa(count), instructionType, filepath.Join(fields...))
			instructionPatches = append(instructionPatches, KustomizePatch{Op: "replace", Path: path, Value: value})
		}
	}
	return instructionPatches, nil
}

// skipElemnt is a helper function for GenericPatchesForSupportBundle - it decides whether or not and
// instruction should be skipped based on whether pathsToSkip and/or lookUpValue exists within the instruction.
func skipElement(element *kyaml.RNode, pathsToSkip [][]string, lookUpValue string) (bool, error) {
	for _, pathToSkip := range pathsToSkip {
		if len(pathToSkip) == 0 {
			continue
		}
		elementNodeToSkip, err := element.Pipe(kyaml.Lookup(pathToSkip...))
		if err != nil {
			return false, err
		}
		if lookUpValue == "" {
			if elementNodeToSkip != nil {
				return true, nil
			}
		} else {
			if strings.TrimSpace(elementNodeToSkip.MustString()) == strings.TrimSpace(lookUpValue) {
				return true, nil
			}
		}
	}
	return false, nil
}

// SpecificPatchForSupportBundle creates and returns KustomizePatch for a kustomiziation file to be applied to the
// SupportBundle.
//
// Inputs:
// * spec: string of the SupportBundle manifest
// * instruction: "collectors" or "analyzers"
// * value: string of Value for patch
// * fields: path of fields (after instruction) to value to be changed in SupportBundle eg {"run","namespace"}
// * lookUpValue: value to compare at path findByFields eg "storageos-operator-logs"
// * findByFields: path of fields to locate the specific instruction
// eg {"logs","name"}
//
// This function is useful in cases where it is desired to set a field such as namespace in a SupportBundle for a
// specific collector or analyzer
func SpecificPatchForSupportBundle(spec, instruction, value string, fields []string, lookUpValue string, findByFields []string) (KustomizePatch, error) {
	kPatch := KustomizePatch{}
	obj, err := kyaml.Parse(spec)
	if err != nil {
		return kPatch, err
	}
	instructionObj, err := obj.Pipe(kyaml.Lookup(
		"spec",
		instruction,
	))
	if err != nil {
		return kPatch, err
	}

	elements, _ := instructionObj.Elements()
	for count, element := range elements {
		if len(findByFields) != 0 {
			elementNodeToPatch, err := element.Pipe(kyaml.Lookup(findByFields...))
			if err != nil {
				return kPatch, err
			}
			if strings.TrimSpace(elementNodeToPatch.MustString()) != strings.TrimSpace(lookUpValue) {
				continue
			}
		}
		path := filepath.Join("/spec", instruction, strconv.Itoa(count), filepath.Join(fields...))
		return KustomizePatch{Op: "replace", Value: value, Path: path}, nil
	}
	return kPatch, fmt.Errorf("path not found in support bundle")
}

// AllInstructionTypesExcept returns [][]string of all instructino types for instruction, except for those provided
func AllInstructionTypesExcept(instruction string, exceptions ...string) ([][]string, error) {
	allTypes, err := getSupportBundleInstructionTypes(instruction)
	if err != nil {
		return nil, err
	}
	finalInstructionTypes := make([][]string, 0)
	for _, instructionType := range allTypes {
		exists := false
		for _, exception := range exceptions {
			if instructionType == exception {
				exists = true
			}
		}
		if exists {
			continue
		}
		single := []string{instructionType}
		finalInstructionTypes = append(finalInstructionTypes, single)
	}

	return finalInstructionTypes, nil
}

// getSupportBundleInstructinoTypes returns the list of types for analyzer or collector instructions
func getSupportBundleInstructionTypes(instruction string) ([]string, error) {
	collectorTypes := []string{
		"clusterInfo",
		"clusterResources",
		"logs",
		"copy",
		"data",
		"secret",
		"run",
		"http",
		"exec",
		"postgresql",
		"mysql",
		"redis",
		"ceph",
		"longhorn",
		"registryImages",
	}
	analyzerTypes := []string{
		"clusterVersion",
		"distribution",
		"containerRuntime",
		"nodeResources",
		"deploymentStatus",
		"statefulsetStatus",
		"imagePullSecret",
		"ingress",
		"storageClass",
		"secret",
		"customResourceDefinition",
		"textAnalyze",
		"postgres",
		"mysql",
		"cephStatus",
		"longhorn",
		"registryImages",
	}

	switch instruction {
	case "collectors":
		return collectorTypes, nil
	case "analyzers":
		return analyzerTypes, nil
	default:
		return nil, fmt.Errorf("unsupported instruction %v, must be \"collectors\" or \"analyzers\"", instruction)
	}

}

// NamespaceYaml returns a yaml string for a namespace object based on the namespace name
func NamespaceYaml(namespace string) string {
	return fmt.Sprintf("%v%v", `apiVersion: v1
kind: Namespace
metadata:
  name: `, namespace)

}
