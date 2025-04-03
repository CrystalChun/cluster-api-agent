package imageregistry

import (
	"encoding/json"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	registryConfKey                = "registries.conf"
	registryCertKey                = "ca-bundle.crt"
	imageRegistryName              = "additional-registry"
	registryCertConfigMapName      = "additional-registry-certificate"
	registryCertConfigMapNamespace = "openshift-config"
	ImageConfigMapName             = "additional-registry-config"
	imageDigestMirrorSetKey        = "image-digest-mirror-set.json"
	imageTagMirrorSetKey           = "image-tag-mirror-set.json"
	registryCertConfigMapKey       = "additional-registry-certificate.json"
	imageConfigKey                 = "image-config.json"
)

type MirrorConfigs struct {
	DigestMirrors      []configv1.ImageDigestMirrors
	TagMirrors         []configv1.ImageTagMirrors
	InsecureRegistries []string
}

// GenerateImageRegistryData generates ConfigMap data for the manifests for mirror registry configuration
// to be created on the spoke cluster. The manifests that are in the data include an ImageDigestMirrorSet CR
// and/or an ImageTagMirrorSet CR with the mirror registry information, a ConfigMap containing the additional
// trusted certificates for the mirror registry, and an image.config.openshift.io CR that references the
// ConfigMap with the additional trusted certificate.
func GenerateImageRegistryData(imageRegistry *corev1.ConfigMap, namespace string) (map[string]string, error) {
	data := make(map[string]string, 0)

	registryConf, ok := imageRegistry.Data[registryConfKey]
	if !ok {
		return nil, fmt.Errorf(
			"failed to find registry key [%s] in configmap %s/%s for image registry configuration",
			registryConfKey,
			imageRegistry.Name,
			namespace,
		)
	}

	mirrorConfigs, err := getImageRegistries(registryConf)
	if err != nil {
		return nil, err
	}

	if len(mirrorConfigs.DigestMirrors) > 0 {
		imageDigestMirrors, err := generateImageDigestMirrorSet(mirrorConfigs.DigestMirrors)
		if err != nil {
			return nil, err
		}
		data[imageDigestMirrorSetKey] = imageDigestMirrors
	}

	if len(mirrorConfigs.TagMirrors) > 0 {
		imageTagMirrors, err := generateImageTagMirrorSet(mirrorConfigs.TagMirrors)
		if err != nil {
			return nil, err
		}
		data[imageTagMirrorSetKey] = imageTagMirrors
	}

	registryCert, registryCertExists := imageRegistry.Data[registryCertKey]
	if registryCertExists {
		registryCertConfigMap, err := generateRegistyCertificateConfigMap(registryCert)
		if err != nil {
			return nil, err
		}
		data[registryCertConfigMapKey] = string(registryCertConfigMap)
	}

	if registryCertExists || len(mirrorConfigs.InsecureRegistries) > 0 {
		imageConfigJSON, err := generateOpenshiftImageConfig(mirrorConfigs.InsecureRegistries, registryCertExists)
		if err != nil {
			return nil, err
		}
		data[imageConfigKey] = string(imageConfigJSON)
	}

	return data, nil
}

// getImageRegistries reads a toml tree string with the structure:
//
// [[registry]]
//
//	 location = "source-registry"
//
//	   [[registry.mirror]]
//		  location = "mirror-registry"
//		  insecure = true # indicates to use an insecure connection to this mirror registry
//		  pull-from-mirror = "tag-only" # indicates to add this to an ImageTagMirrorSet
//
// It will convert it to an ImageDigestMirrorSet CR and/or an ImageTagMirrorSet.
// It will return these as marshalled JSON strings, and it will return a string list of insecure
// mirror registries (if they exist)
func getImageRegistries(registry string) (*MirrorConfigs, error) {
	mirrorConfigs := &MirrorConfigs{
		DigestMirrors:      make([]configv1.ImageDigestMirrors, 0),
		TagMirrors:         make([]configv1.ImageTagMirrors, 0),
		InsecureRegistries: make([]string, 0),
	}
	// Parse toml and add mirror registry to image digest mirror set
	tomlTree, err := toml.Load(registry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load value of registries.conf into toml tree; incorrectly formatted toml")
	}
	registriesTree, ok := tomlTree.Get("registry").([]*toml.Tree)
	if !ok {
		return nil, fmt.Errorf("failed to find registry key in toml tree")
	}

	// Add each registry and its mirrors as either an image digest mirror or an image tag mirror
	for _, registry := range registriesTree {
		source, sourceExists := registry.Get("location").(string)
		mirrorTrees, mirrorExists := registry.Get("mirror").([]*toml.Tree)
		if !sourceExists {
			return nil, fmt.Errorf("failed to parse image registry toml, missing registry location")
		}
		if !mirrorExists {
			return nil, fmt.Errorf("failed to parse image registry toml, missing registry.mirror in registries.conf")
		}
		if err := parseMirrorRegistries(mirrorConfigs, mirrorTrees, source); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse image mirror registry toml for registry %s", source))
		}
	}

	if len(mirrorConfigs.DigestMirrors) < 1 && len(mirrorConfigs.TagMirrors) < 1 {
		return nil, fmt.Errorf("failed to find any image mirrors in registry.conf")
	}
	return mirrorConfigs, nil
}

// parseMirrorRegistries takes a mirror registry toml tree and parses it into
// a list of image mirrors and a string list of insecure mirror registries.
func parseMirrorRegistries(
	mirrorConfigs *MirrorConfigs,
	mirrorTrees []*toml.Tree,
	source string,
) error {
	itmsMirror := configv1.ImageTagMirrors{}
	idmsMirror := configv1.ImageDigestMirrors{}
	for _, mirrorTree := range mirrorTrees {
		mirror, ok := mirrorTree.Get("location").(string)
		if !ok {
			return fmt.Errorf("missing mirror registry location key or the value of location is not a string")
		}
		if insecure, ok := mirrorTree.Get("insecure").(bool); ok && insecure {
			mirrorConfigs.InsecureRegistries = append(mirrorConfigs.InsecureRegistries, mirror)
		}
		if pullFrom, ok := mirrorTree.Get("pull-from-mirror").(string); ok && pullFrom == "tag-only" {
			itmsMirror.Source = source
			itmsMirror.Mirrors = append(itmsMirror.Mirrors, configv1.ImageMirror(mirror))
			continue
		}
		idmsMirror.Source = source
		idmsMirror.Mirrors = append(idmsMirror.Mirrors, configv1.ImageMirror(mirror))
	}
	if len(itmsMirror.Mirrors) > 0 {
		mirrorConfigs.TagMirrors = append(mirrorConfigs.TagMirrors, itmsMirror)
	}
	if len(idmsMirror.Mirrors) > 0 {
		mirrorConfigs.DigestMirrors = append(mirrorConfigs.DigestMirrors, idmsMirror)
	}
	return nil
}

func generateImageDigestMirrorSet(mirrors []configv1.ImageDigestMirrors) (string, error) {
	idms, err := json.Marshal(configv1.ImageDigestMirrorSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageDigestMirrorSet",
			APIVersion: configv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: imageRegistryName,
		},
		Spec: configv1.ImageDigestMirrorSetSpec{
			ImageDigestMirrors: mirrors,
		},
	})
	return string(idms), err
}

func generateImageTagMirrorSet(mirrors []configv1.ImageTagMirrors) (string, error) {
	itms, err := json.Marshal(configv1.ImageTagMirrorSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageTagMirrorSet",
			APIVersion: configv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: imageRegistryName,
		},
		Spec: configv1.ImageTagMirrorSetSpec{
			ImageTagMirrors: mirrors,
		},
	})
	return string(itms), err
}

// generateRegistyCertificateConfigMap creates a ConfigMap CR containing the registry certificate
// to be created on the spoke cluster and marshals the ConfigMap to JSON
func generateRegistyCertificateConfigMap(certificate string) ([]byte, error) {
	// Add certificate as a ConfigMap manifest to be created in the spoke cluster's openshift-config namespace
	certificateCM := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      registryCertConfigMapName,
			Namespace: registryCertConfigMapNamespace,
		},
		Data: map[string]string{
			registryCertKey: certificate,
		},
	}
	return json.Marshal(certificateCM)
}

// generateOpenshiftImageConfig creates an image.config.openshift.io CR that references the additional
// regitry certificate ConfigMap CR if it exists and sets any insecure registries. It then marshals
// it into JSON.
func generateOpenshiftImageConfig(insecureRegistries []string, addRegistryCert bool) ([]byte, error) {
	clusterImageConfig := &configv1.Image{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Image",
			APIVersion: configv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	if addRegistryCert {
		clusterImageConfig.Spec.AdditionalTrustedCA.Name = registryCertConfigMapName
	}
	if len(insecureRegistries) > 0 {
		clusterImageConfig.Spec.RegistrySources.InsecureRegistries = insecureRegistries
	}
	return json.Marshal(clusterImageConfig)
}
