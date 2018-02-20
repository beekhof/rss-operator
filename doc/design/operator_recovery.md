# Operator Recovery

- Create CRD
 - If the creation succeed, then the operator is a new one and does not require recovery. END.
- Find all existing clusters
 - loop over the custom resource definition items to get all created clusters
- Reconstruct clusters
 - for each cluster, find running Pods that belong to it by label selection
 - recover the membership by issuing member list call
 - check application status with the configured `status` command on each Pod
 - recover sequence numbers by calling the configured `sequence` command on each Pod
