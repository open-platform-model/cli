// Package initvalues is a test module that declares both initValues and
// debugValues with different content, so a test can show which one an
// instance package starts from and how non-concrete initValues render.
package initvalues

import (
	"strings"

	m "opmodel.dev/core@v2"
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"

	id "test.example/modules/initvalues/identity"
)

m.#Module

metadata: {
	_segments:   strings.Split(strings.SplitN(id.ModulePath, "@", 2)[0], "/")
	name:        _segments[len(_segments)-1]
	modulePath:  id.ModulePath
	version:     id.Version
	description: "initValues and debugValues side by side"
}

#config: {
	image: res.#Image & {
		repository: string | *"nginx"
		tag:        string | *"1.29"
		digest:     string | *""
	}
	replicas: int & >=1 | *1
	logLevel: "info" | "debug" | *"info"
	port:     int & >0 & <=65535 | *80
}

// initValues: a defaulted field, an undefaulted disjunction the user must
// pick from, and an optional field.
initValues: {
	replicas: *2 | int
	logLevel: "info" | "debug"
	port?:    int
}

// debugValues: concrete, and different from initValues throughout.
debugValues: {
	image: {
		repository: "debug.example/nginx"
		tag:        "debug"
		digest:     ""
	}
	replicas: 7
	logLevel: "debug"
	port:     8080
}

#components: {
	app: {
		bp.#StatelessWorkload

		metadata: name: "app"

		spec: statelessWorkload: {
			container: {
				name:  "app"
				image: #config.image
				ports: http: {
					name:       "http"
					targetPort: #config.port
				}
			}
			scaling: count: #config.replicas
			restartPolicy: "Always"
			updateStrategy: type: "RollingUpdate"
		}
	}
}
