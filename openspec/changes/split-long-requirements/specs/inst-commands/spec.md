## REMOVED Requirements

### Requirement: instance cluster-query commands accept an instance identifier as positional argument
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file", "A path identifier whose package is not a valid instance is refused with exit code 1".

### Requirement: instance render commands accept --platform flag
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source", "The platform is not specified in the instance file and there is no `--provider` flag".

### Requirement: instance render commands refuse a namespace override that disagrees with the instance file
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`", "The namespace refusal names the source of the override, the override value, the instance's namespace and the instance argument", "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override", "The `-n` help text of `opm instance apply`, `build`, `diff` and `vet` states the constraint on the value", "The module commands are unaffected by the namespace refusal of the instance render commands".

### Requirement: Instance diff fails when a rendered resource could not be read
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm instance diff` treats a rendered resource it could not read or compare as a failure", "`opm instance diff` exits non-zero and prints no `No differences found` when a failure exists", "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource".

## ADDED Requirements

### Requirement: Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape
The cluster-query commands (`status`, `tree`, `events`, `delete`) under `opm instance` SHALL accept an instance identifier as a required positional argument. The CLI SHALL auto-detect whether the identifier is a path, a UUID or an instance name, from the identifier's shape alone and never from what exists on disk.

#### Scenario: status by instance name

- **WHEN** `opm instance status jellyfin` is run
- **THEN** the CLI SHALL look up the instance by a direct `ModuleInstance` GET by name
- **AND** display the instance health status

#### Scenario: status by UUID

- **WHEN** `opm instance status 550e8400-e29b-41d4-a716-446655440000` is run
- **THEN** the CLI SHALL look up the instance by listing `ModuleInstance` CRs and matching `status.instanceUUID`
- **AND** display the instance health status

#### Scenario: delete by instance name

- **WHEN** `opm instance delete jellyfin` is run
- **THEN** the CLI SHALL look up the instance by name and delete its resources from the cluster

#### Scenario: identifier auto-detection

- **WHEN** the positional argument ends with `.cue`, contains `/` or the OS path separator, or starts with `.` or `~`
- **THEN** the CLI SHALL treat it as a path
- **WHEN** the positional argument matches the pattern `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
- **THEN** the CLI SHALL treat it as a UUID
- **WHEN** the positional argument is neither
- **THEN** the CLI SHALL treat it as an instance name

#### Scenario: a bare word is a name even when a directory of that name exists

- **WHEN** `opm instance status hello` is run in a directory that contains a `hello` directory
- **THEN** the CLI SHALL look up the instance named `hello`
- **AND** SHALL NOT read the `hello` directory
- **WHEN** `opm instance status ./hello` is run in the same directory
- **THEN** the CLI SHALL treat `./hello` as a path

### Requirement: A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file
A path (an `instance.cue` file, or a directory holding one) SHALL be acquired through the library kernel as an instance package with the invocation's resolved registry, and the instance name and namespace SHALL be read from the acquired instance's metadata; the `--namespace` flag SHALL take precedence over the namespace in the file.

#### Scenario: status by instance path

- **WHEN** `opm instance status ./jellyfin/instance.cue` is run and the package declares `metadata.name: "jellyfin"` and `metadata.namespace: "media"`
- **THEN** the CLI SHALL acquire the package through the kernel and look up the instance named `jellyfin` in the `media` namespace

#### Scenario: namespace flag overrides the file

- **WHEN** `opm instance status ./jellyfin -n staging` is run and the package declares `metadata.namespace: "media"`
- **THEN** the CLI SHALL look up the instance in the `staging` namespace

### Requirement: A path identifier whose package is not a valid instance is refused with exit code 1
A path whose package is not a valid instance (wrong kind, no concrete `metadata.name` or `metadata.namespace`, build failure) SHALL be refused with exit code 1 and an error naming the path.

#### Scenario: path that is not a valid instance is refused

- **WHEN** `opm instance status ./jellyfin` is run and the package's `metadata.namespace` is not concrete, or its `kind` is not `ModuleInstance`
- **THEN** the CLI SHALL exit with code 1 and an error naming the path and the refusal

### Requirement: Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source
Render commands (`vet`, `build`, `apply`, `diff`) SHALL accept a `--platform <dir>` flag naming a platform module directory, the highest-precedence platform source (see the `platform-resolution` capability), through the shared `cmdutil.InstanceFileFlags` set alongside `-f`/`--values`. The flag help SHALL say it overrides the cluster Platform and the instance's own deps.

#### Scenario: platform flag overrides the resolved platform source

- **WHEN** `opm instance build instance.cue --platform ./my-platform` is run
- **THEN** the render SHALL use the platform module in `./my-platform` regardless of the cluster Platform CR or the instance's deps

### Requirement: The platform is not specified in the instance file and there is no `--provider` flag
The platform SHALL NOT be specified in the instance file, and there SHALL be no `--provider` flag: it was retired together with the provider concept. (Requirement: Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source)

#### Scenario: There is no `--provider` flag on the render commands

- **WHEN** a render command (`vet`, `build`, `apply` or `diff`) under `opm instance` is run
- **THEN** the platform SHALL NOT be specified in the instance file, and there SHALL be no `--provider` flag: it was retired together with the provider concept

### Requirement: `opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`
The instance file owns its namespace: `metadata.namespace` is part of the instance's identity. When `opm instance apply`, `build`, `diff` or `vet` resolves its namespace from the `-n`/`--namespace` flag or the `OPM_NAMESPACE` environment variable, and that value differs from the acquired instance's `metadata.namespace`, the command SHALL refuse with exit code 2 before resolving the platform, rendering, diffing or applying anything.

#### Scenario: Flag differs from the file's namespace

- **WHEN** `opm instance apply ./jellyfin -n staging` is run and the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL exit with code 2 and apply nothing
- **AND** the error SHALL name `--namespace`, `staging`, `media` and `./jellyfin`, and tell the user to edit `metadata.namespace` in the instance file

### Requirement: The namespace refusal names the source of the override, the override value, the instance's namespace and the instance argument
The refusal SHALL name the source of the override (`--namespace` or `OPM_NAMESPACE`), the override value, the instance's `metadata.namespace` value and the instance argument, and SHALL tell the user to edit `metadata.namespace` in the instance file to deploy the instance elsewhere. (Requirement: `opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`)

#### Scenario: Environment variable differs from the file's namespace

- **WHEN** `OPM_NAMESPACE=staging opm instance build ./jellyfin` is run without `-n` and the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL exit with code 2 and print no manifests
- **AND** the error SHALL name `OPM_NAMESPACE`, `staging` and `media`

### Requirement: An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override
An override equal to `metadata.namespace` SHALL be accepted. A namespace that comes from `kubernetes.namespace` in `~/.opm/config.cue` or from the built-in default is not an override and SHALL NOT be compared; the instance renders in its file's namespace. (Requirement: `opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`)

#### Scenario: Flag equal to the file's namespace

- **WHEN** `opm instance apply ./jellyfin -n media` is run and the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL render and apply the instance in `media` as without the flag

#### Scenario: Flag matches while OPM_NAMESPACE differs

- **WHEN** `OPM_NAMESPACE=staging opm instance apply ./jellyfin -n media` is run and the instance declares `metadata.namespace: "media"`
- **THEN** the flag SHALL win and the command SHALL NOT refuse: the shadowed `OPM_NAMESPACE` value is not compared

#### Scenario: No override

- **WHEN** `opm instance diff ./jellyfin` is run with neither `-n` nor `OPM_NAMESPACE`, and `~/.opm/config.cue` sets `kubernetes.namespace: "staging"`
- **AND** the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL NOT refuse, and SHALL diff the instance in `media`

### Requirement: The `-n` help text of `opm instance apply`, `build`, `diff` and `vet` states the constraint on the value
The `-n` help text of these four commands SHALL state that the value must equal the instance file's `metadata.namespace`. (Requirement: `opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`)

#### Scenario: Help text states the constraint

- **WHEN** `opm instance apply --help`, `opm instance build --help`, `opm instance diff --help` or `opm instance vet --help` is run
- **THEN** the `-n, --namespace` line SHALL say the value must equal the instance file's `metadata.namespace`

### Requirement: The module commands are unaffected by the namespace refusal of the instance render commands
The module commands (`opm module build`, `apply`, `vet`) SHALL be unaffected: they synthesize the instance in the override namespace, so the override and `metadata.namespace` agree. (Requirement: `opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`)

#### Scenario: Module path still honours the override

- **WHEN** `opm module apply ./my-module -n staging` is run
- **THEN** the synthesized instance's `metadata.namespace` SHALL be `staging` and the command SHALL NOT refuse

### Requirement: `opm instance diff` treats a rendered resource it could not read or compare as a failure
`opm instance diff` SHALL treat a rendered resource whose live read fails with an error other than NotFound, and a rendered resource whose comparison with the live object fails, as a failure. For each failure the command SHALL print an error that names the resource and carries the cause. The command SHALL still compare the resources it can read and SHALL print the differences it found.

#### Scenario: One unreadable resource beside a modified one

- **WHEN** one rendered resource differs from its live object
- **AND** the read of another rendered resource fails with an internal server error
- **THEN** the command SHALL print the difference of the first resource
- **AND** SHALL print an error naming the second resource
- **AND** SHALL exit 1

### Requirement: `opm instance diff` exits non-zero and prints no `No differences found` when a failure exists
When at least one failure exists, the `opm instance diff` command SHALL NOT print `No differences found` and SHALL exit non-zero: with the code of the failures when all of them have the same class (4 when the API server denied the read with Forbidden or Unauthorized, 3 on a server timeout or service unavailable, 1 for any other failure), and 1 when the classes differ. (Requirement: `opm instance diff` treats a rendered resource it could not read or compare as a failure)

#### Scenario: Every rendered resource is unreadable

- **WHEN** `opm instance diff` runs and the read of every rendered resource fails with Forbidden
- **THEN** the command SHALL print one error per resource naming it
- **AND** SHALL NOT print `No differences found`
- **AND** SHALL exit 4

#### Scenario: Failures of two classes

- **WHEN** one read fails with Forbidden and another with an internal server error
- **THEN** the command SHALL exit 1

### Requirement: `opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource
A NotFound answer for a rendered resource SHALL still mean a new resource. With no failure the `opm instance diff` command SHALL behave as before. A tracked resource that orphan detection could not read stays a warning that does not change the exit code.

#### Scenario: Unreadable tracked resource that is not rendered

- **WHEN** every rendered resource is read
- **AND** orphan detection cannot read a tracked resource that is not rendered
- **THEN** the command SHALL warn that orphan detection could not check it
- **AND** the exit code SHALL NOT change

#### Scenario: Nothing failed and nothing differs

- **WHEN** every rendered resource is read and equals its live object
- **THEN** the command SHALL print `No differences found` and exit 0
