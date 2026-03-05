// This file adds a "values" field at package root, NOT inside #Module.
// Question: Does this break importability when assigned to a #module field?
package withvalues

values: {
	image: {
		repository: "nginx"
		tag:        "1.21"
	}
	replicas: 3
}
