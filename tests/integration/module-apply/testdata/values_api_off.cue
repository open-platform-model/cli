// values_api_off.cue: drop the `api` component from the rendered set so the
// integration test can verify pruning when `opm module apply` is re-run.
package itest

values: {
	web: {
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
		port:    8080
	}
	api: {
		enabled: false
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
		port:    3000
	}
}
