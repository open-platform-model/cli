package schemas

/////////////////////////////////////////////////////////////////
//// Security Schemas
/////////////////////////////////////////////////////////////////

#WorkloadIdentitySchema: {
	name!:           string
	automountToken?: bool
}

// Security context constraints for container and pod-level hardening
#SecurityContextSchema: {
	// Run container as non-root user
	runAsNonRoot: bool
	// Specific user ID to run as
	runAsUser?: int
	// Specific group ID to run as
	runAsGroup?: int
	// Group ID applied to all mounted volumes (pod-level; makes volumes writable by the group)
	fsGroup?: int
	// Mount the root filesystem as read-only
	readOnlyRootFilesystem: bool
	// Prevent privilege escalation
	allowPrivilegeEscalation: bool
	// Linux capabilities
	capabilities?: {
		add?: [...string]
		drop: [...string] | ["ALL"]
	}
}

// Standalone service account identity
#ServiceAccountSchema: {
	name!:           string
	automountToken?: bool
}

// Single RBAC permission rule
#PolicyRuleSchema: {
	apiGroups!: [...string]
	resources!: [...string]
	verbs!: [...string]
}

// Role subject — embeds an identity directly via CUE reference
#RoleSubjectSchema: {#WorkloadIdentitySchema | #ServiceAccountSchema}

// RBAC role with rules and CUE-referenced subjects
#RoleSchema: {
	name!: string
	scope: *"namespace" | "cluster"
	rules!: [...#PolicyRuleSchema] & [_, ...]
	subjects!: [...#RoleSubjectSchema] & [_, ...]
}

#EncryptionConfigSchema: {
	atRest:    bool
	inTransit: bool
}
