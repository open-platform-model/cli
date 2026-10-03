## MODIFIED Requirements

### Requirement: Tree sorts resources within components by weight

Within each component group, resources SHALL be sorted by OPM weight (ascending) and then alphabetically by name. This ensures tree output matches apply order. The tree SHALL sort explicitly and SHALL NOT rely on the order in which the inventory stores its entries, which follows render order.

#### Scenario: Resources sorted by weight

- **WHEN** a component contains a Deployment (weight 100), a Service (weight 50), and a ConfigMap (weight 15)
- **THEN** the tree SHALL display ConfigMap, then Service, then Deployment (ascending weight order)

#### Scenario: Resources with equal weight sorted by name

- **WHEN** a component contains two ConfigMaps named `config-a` and `config-z` (both weight 15)
- **THEN** the tree SHALL display `config-a` before `config-z` (alphabetical)

#### Scenario: Inventory order does not decide the tree order

- **WHEN** the inventory lists a component's Deployment before its ConfigMap
- **THEN** the tree SHALL still display the ConfigMap before the Deployment
