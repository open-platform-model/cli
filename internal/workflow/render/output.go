package render

import (
	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/output"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseManifestOutputFormat(outputFmt string) (output.Format, error) {
	return cmdutil.ParseManifestOutputFormat(outputFmt)
}

func WriteManifestOutput(resources []*unstructured.Unstructured, outputFormat output.Format, split bool, outDir, instanceName string) error {
	return cmdutil.WriteManifestOutput(resources, outputFormat, split, outDir, instanceName)
}
