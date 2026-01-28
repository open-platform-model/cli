package core

// #CompiledModule: The compiled and optimized form (IR - Intermediate Representation)
// Result of flattening a Module:
//   - Blueprints expanded into their constituent Resources, Traits, and Policies
//   - Structure optimized for runtime evaluation
//   - Ready for binding with concrete values
// May include platform additions (Policies, Scopes, Components) if created from
// a platform team's extended Module, but primary purpose is compilation
#CompiledModule: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "CompiledModule"

	metadata: #Module.metadata

	// Components (with blueprints expanded)
	#components: [string]: #Component

	// Scopes (from Module)
	#scopes?: [Id=string]: #Scope

	// Value schema (preserved from Module)
	#spec: _

	// Concrete values (preserved from Module)
	values: _

	#status?: {
		componentCount: len(#components)
		scopeCount?: {if #scopes != _|_ {len(#scopes)}}
		...
	}
})

#CompiledModuleMap: [string]: #CompiledModule
