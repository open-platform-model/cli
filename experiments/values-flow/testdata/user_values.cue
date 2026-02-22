// user_values.cue — simulates a --values flag file (Layer 2 override).
//
// Values are deliberately different from both fixture defaults:
//   values_module defaults: image="nginx:latest",  replicas=1
//   inline_module defaults: image="nginx:stable",  replicas=2
//   user_values (this):     image="custom:2.0",    replicas=5
//
// This makes it unambiguous in assertions which source won.
// No package declaration — values files may be package-free.
values: {
	image:    "custom:2.0"
	replicas: 5
}
