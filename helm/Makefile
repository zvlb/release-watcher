URL=https://zvlb.github.io/release-watcher/helm
HELM_IMAGE = alpine/helm:3.9.1
HELM_DOCS_IMAGE = jnorwood/helm-docs:v1.11.0
KNOWN_TARGETS=helm

# Run linter for helm chart
lint:
	helm lint charts/release-watcher

# Package chart into zip file
package:
	helm package charts/* -d packages

# Create index file (use only for initial setup)
index:
	helm repo index --url ${URL} .
