# Agentic Foundry Infrastructure (GCP Terraform Module)

This directory contains the official Google Cloud Platform (GCP) Terraform module to provision and manage the cloud infrastructure for **Agentic Foundry**.

## Architecture Overview

The Terraform module (`infra/modules/`) provisions:
- **Serverless Compute**: Cloud Run services for UI Frontend, Go API Backend, and AI Agent Engine.
- **Background Jobs**: Cloud Run v2 Job and Cloud Scheduler for nightly workspace maintenance.
- **Networking**: VPC Network, Cloud NAT, Serverless VPC Access Connector, and Cloud DNS records.
- **Load Balancing & Security**: HTTPS Global External Application Load Balancer with Identity-Aware Proxy (IAP) integration and SSL certificates.
- **Data & Storage**:
  - Cloud SQL PostgreSQL instance for session management.
  - Google Cloud Storage (GCS) buckets for workspaces, config, and artifacts.
  - Cloud Firestore database for application metadata.
  - Google Cloud Filestore (NFS) for shared agent workspace persistence.
- **Observability & Secrets**: Google Secret Manager, Cloud Logging, Cloud Trace, and OpenTelemetry monitoring.

## Directory Structure

```
infra/
├── local/               # Local Docker Compose setup for development PostgreSQL & pgAdmin
├── cicd/                # CI/CD delivery pipelines & deployment configurations
│   ├── base/            # Core deployment pipeline (Cloud Build, DeliveryPipelines, Skaffold, Manifests)
│   └── environments/    # Environment target configurations and deploy parameters (gitignored)
├── modules/             # Reusable GCP Terraform infrastructure module
└── examples/
    └── complete/        # Working example showing how to invoke the Terraform module
```

## Quick Start

### 1. Prerequisites
- [Google Cloud SDK (`gcloud`)](https://cloud.google.com/sdk/docs/install)
- [Terraform >= 1.5.0](https://developer.hashicorp.com/terraform/downloads)
- A GCP project with billing enabled and OAuth 2.0 Client ID configured for IAP.

### 2. Deployment

1. Navigate to the complete example:
   ```bash
   cd infra/examples/complete
   ```
2. Create and configure your `terraform.tfvars`:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your GCP project ID, region, domain, and IAP client ID
   ```
3. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `project_id` | Google Cloud Project ID | `string` | n/a | yes |
| `project_number` | Google Cloud Project Number | `string` | n/a | yes |
| `region` | GCP region for regional resources | `string` | `"us-central1"` | no |
| `zone` | GCP zone for zonal resources (e.g. Filestore) | `string` | `"us-central1-c"` | no |
| `prefix` | Resource naming prefix | `string` | `"apf"` | no |
| `domain_name` | Custom domain name for the environment | `string` | n/a | yes |
| `iap_client_id` | OAuth 2.0 Client ID for Identity-Aware Proxy | `string` | n/a | yes |
| `iap_allowed_members` | List of IAM principals/domains allowed to access IAP | `list(string)` | `[]` | no |
| `subnet_range` | CIDR range for the primary VPC subnet | `string` | `"10.10.0.0/24"` | no |
| `vpc_connector_range` | CIDR range for the VPC access connector | `string` | `"10.10.1.0/28"` | no |

## Outputs

| Name | Description |
|------|-------------|
| `ui_url` | URL endpoint of the UI Frontend |
| `api_url` | URL endpoint of the API Backend |
| `ai_url` | URL endpoint of the AI Agent Engine |
| `load_balancer_ip` | External Load Balancer IP Address |
