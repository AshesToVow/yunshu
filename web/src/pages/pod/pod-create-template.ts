export const POD_CREATE_YAML_TEMPLATE = `apiVersion: v1
kind: Pod
metadata:
  name: demo-pod
spec:
  containers:
  - name: main
    image: nginx:latest
`;
