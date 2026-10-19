## REMOVED Requirements

### Requirement: Public resource ordering API
**Reason**: The weight table moved to the library's Kubernetes tier, `opm/k8s/object`, so both frontends order by one definition (enhancement 0012, "Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle"). The cli deletes its copy.
**Migration**: Use `object.Weight` from `github.com/open-platform-model/library/opm/k8s/object`; the values are unchanged. The cli's ordering rule is the `resource-conversion` requirement "Object order comes from the library weight table".

### Requirement: No CLI dependencies
**Reason**: The package is deleted, so there is no dependency tree to constrain. The library tier's own lint rules fence `opm/k8s/object`.
**Migration**: None.

### Requirement: One stable weight sort
**Reason**: The stable sort moved with the table to `opm/k8s/object.Sort`.
**Migration**: Use `object.Sort`, `object.Direction`, `object.Ascending` and `object.Descending`. The rule that every cli ordering path uses it is the `resource-conversion` requirement "Object order comes from the library weight table".
