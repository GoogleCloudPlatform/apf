# Cloud Deploy Environment Targets & Configurations

This directory manages **Google Cloud Deploy Target** definitions and their environment-specific deployment parameters (`deployParameters`).

## Architecture & Security Model

The deployment pipeline is strictly split into two layers:
1. **Core Pipeline (`infra/cicd/base/`)**: Public and generic definitions for delivery pipelines, Skaffold build specs, and Cloud Run manifests.
2. **Environment Targets (`infra/cicd/environments/`)**: Target environments containing concrete infrastructure identifiers (GCP project IDs, Cloud SQL instances, VPC networks, IAP audiences, secret names, custom domains).

> **Security Notice**:
> Live environment definitions (e.g. `staging.yaml`, `production.yaml`) contain project-specific infrastructure parameters and are **ignored by Git** (`.gitignore`).
> For production and internal deployments, store these target definitions in your private infrastructure repository or apply them via your secure CI/CD orchestration.

---

## Configuring an Environment Target

To configure a target environment for your GCP project:

1. Copy the example template:
   ```bash
   cp infra/cicd/environments/targets.example.yaml infra/cicd/environments/staging.yaml
   ```
2. Replace all placeholder values in `staging.yaml` with your GCP project infrastructure parameters:
   - `project_id`: Your Google Cloud Project ID
   - `cloud_location`: Region for resources (e.g. `us-central1`)
   - `service_account`: Dedicated runtime service account email
   - `api_backend`, `app_url`, `app_external_url`: Endpoints and domains
   - `cloudsql_*`: Database connection name and credentials
   - `artifact_bucket`: GCS bucket for artifact storage
   - `nfs_server`, `nfs_path`: Filestore NFS endpoint
   - `iap_audience`: Identity-Aware Proxy OAuth Client / Backend Service ID
3. Apply the target definition to Google Cloud Deploy:
   ```bash
   gcloud deploy apply --file=infra/cicd/environments/staging.yaml --region=us-central1
   ```

Once applied, Cloud Deploy persists the target configuration in the GCP control plane. The core build pipeline (`infra/cicd/base/cloudbuild.yaml`) can now create and deploy releases to this target without needing any local target files in the application repository.
