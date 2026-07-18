// values_api_off.cue: drop the `api` component from the rendered set so the
// integration test can verify pruning when `opm module apply` is re-run.
// Lives OUTSIDE the module package directory: kernel LoadModulePackage loads
// every package file in the module dir, and a stray `values` field would
// collide with the closed #Module shape.
values: {
	web: {
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
	}
	api: {
		enabled: false
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
	}
}
