## MODIFIED Requirements

### Requirement: Tree sorts resources within components by weight

Within each component group, resources SHALL be sorted by OPM weight (ascending) and then alphabetically by name, with kind, API group and then namespace breaking any remaining tie, so the same inventory always prints in the same order. This ensures tree output matches apply order at weight granularity. The tree SHALL sort when it builds the groups; it SHALL NOT rely on the order of the inventory entries, which follow the render order of the last apply.

#### Scenario: Resources sorted by weight

- **WHEN** a component contains a Deployment (weight 100), a Service (weight 50), and a ConfigMap (weight 15)
- **THEN** the tree SHALL display ConfigMap, then Service, then Deployment (ascending weight order)

#### Scenario: Resources with equal weight sorted by name

- **WHEN** a component contains two ConfigMaps named `config-a` and `config-z` (both weight 15)
- **THEN** the tree SHALL display `config-a` before `config-z` (alphabetical)

#### Scenario: Inventory order does not decide the tree order

- **WHEN** the inventory of the last apply lists a component's Deployment `web`, Service `web`, ConfigMap `config-z` and ConfigMap `config-a` in that order
- **THEN** the tree SHALL display ConfigMap `config-a`, ConfigMap `config-z`, Service `web`, Deployment `web`, in the text, JSON and YAML outputs alike
