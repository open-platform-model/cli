package render

import (
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/output"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseManifestOutputFormat(outputFmt string) (output.Format, error) {
	return cmdutil.ParseManifestOutputFormat(outputFmt)
}

func WriteManifestOutput(resources []*unstructured.Unstructured, outputFormat output.Format, split bool, outDir, releaseName string) error {
	return cmdutil.WriteManifestOutput(resources, outputFormat, split, outDir, releaseName)
}
