# Deployment Checklist & Activities for Agentic Foundry

This document outlines the activities, requirements, and configurations needed to deploy the Agentic Product Foundry application into a completely new Google Cloud Platform (GCP) environment.

---

## 1. Authentication Mechanism (IAP-Based Boundary Security)

The application relies on Google Cloud **Identity-Aware Proxy (IAP)** to protect the entry-point (UI frontend service) and verify user identity. 

### Pre-requisites & Activities:
1. **Domain Name Registration**:
   - Register or allocate a custom domain (e.g. `apf.yourdomain.com`).
2. **OAuth Consent Screen Setup**:
   - Navigate to **APIs & Services > OAuth Consent Screen** in the GCP Console.
   - Configure the consent screen as **Internal** (limiting access to your organization's Google Workspace / Cloud Identity users).
3. **Generate OAuth 2.0 Credentials**:
   - Navigate to **APIs & Services > Credentials**.
   - Create an **OAuth Client ID** of type **Web Application**.
   - Note the generated **Client ID** and **Client Secret**.
4. **Secret Manager Provisioning**:
   - Create a secret in Secret Manager named `iap-client-secret` containing the OAuth 2.0 Client Secret value.
   - Ensure the Compute Service Account has `roles/secretmanager.secretAccessor` permissions to access it.
5. **IAM Configuration**:
   - Grant access to authorized users/groups by assigning the `roles/iap.httpsResourceAccessor` role on the Load Balancer backend service (e.g., domain-wide access like `domain:yourdomain.com`).
6. **IAP Verification in Backend API**:
   - The UI and Backend API check and parse the following headers passed by IAP:
     - `x-goog-authenticated-user-email`: Identifies the user's email address.
     - `x-iap-jwt-assertion`: JWT token used for authenticating subsequent/internal calls.

---

## 2. GCP Services (APIs) to be Enabled

The following GCP Services/APIs must be enabled in the target project before executing Terraform:

| Service / API Name | Identifier | Purpose |
| :--- | :--- | :--- |
| **Compute Engine API** | `compute.googleapis.com` | VPC networks, Subnets, Cloud NAT, Cloud Router, and Cloud Load Balancing. |
| **Service Networking API** | `servicenetworking.googleapis.com` | Private VPC Connection (peering) for Cloud SQL PostgreSQL. |
| **Vertex AI API** | `aiplatform.googleapis.com` | Machine learning services, Vertex AI Studio, and model registry. |
| **Generative Language API** | `generativelanguage.googleapis.com` | Access to Gemini models. |
| **Vertex AI Agent Builder API** | `discoveryengine.googleapis.com` | LLM-based search and agent capabilities. |
| **Cloud Storage API** | `storage-component.googleapis.com` | Configuration and artifact storage. |
| **Cloud Filestore API** | `file.googleapis.com` | NFS server hosting agent workspaces. |
| **Cloud DNS API** | `dns.googleapis.com` | Private and public DNS record routing. |
| **Cloud KMS API** | `cloudkms.googleapis.com` | Encryption keys for data security. |
| **Cloud Run API** | `run.googleapis.com` | Serving containerized UI, API, and AI Agent services. |
| **Cloud Build API** | `cloudbuild.googleapis.com` | Running image build and compilation pipelines. |
| **Cloud Deploy API** | `deploy.googleapis.com` | Delivery pipelines and release management. |
| **Secret Manager API** | `secretmanager.googleapis.com` | Storage of sensitive database passwords and API tokens. |
| **Cloud SQL Admin API** | `sqladmin.googleapis.com` | Managing the Cloud SQL PostgreSQL instance. |
| **Cloud Trace API** | `cloudtrace.googleapis.com` | Request tracing and latency troubleshooting. |
| **Firestore API** | `firestore.googleapis.com` | Document storage and messaging services. |
| **Identity-Aware Proxy API** | `iap.googleapis.com` | User authentication at the gateway. |
| **Artifact Registry API** | `artifactregistry.googleapis.com` | Managing Docker container images. |
| **Resource Manager API** | `cloudresourcemanager.googleapis.com` | Managing IAM and workspace projects. |

---

## 3. Customer Subscriptions and External Credentials

The customer must provide or establish the following accounts, subscriptions, and credentials:

1. **Google Workspace / Cloud Identity Subscription**:
   - For managing users, organization domains, and single-sign-on (SSO) auth policies mapped to IAP.
2. **Domain Hosting Provider / Cloud DNS**:
   - Access to modify DNS records (CNAME, A records) to point the custom domain name to the External Load Balancer IP.
3. **GitHub Account & Integration Config**:
   - Generate a GitHub Personal Access Token (PAT) with repository scopes to integrate workspaces.

---

## 4. Security, Networking, Compliance, and Monitoring Controls

### Networking
- **VPC Infrastructure**:
  - A custom VPC (`*-vpc`) with a dedicated subnetwork for the compute region (e.g. `10.11.0.0/24`).
  - **Direct VPC Egress**: Configured on Cloud Run services to route all egress traffic directly into the custom VPC subnetwork, enabling secure access to internal resources.
  - **Cloud NAT** and **Cloud Router**: Ensures instances in the private network can query external APIs (e.g. GitHub, external DBs) without possessing public IP addresses.
- **Ingress Isolation**:
  - **UI Service**: Ingress set to `internal-and-cloud-load-balancing` (accessible only via Cloud HTTPS Load Balancer protected by IAP).
  - **API & AI Services**: Ingress set to `internal` (strictly blocked from public endpoints; only reachable within the VPC).
- **Internal DNS Configuration**:
  - Private DNS Managed Zone for `run.app` and `googleapis.com` pointing to `restricted.googleapis.com` IP ranges (`199.36.153.4` - `199.36.153.7`). This prevents network calls from traversing the public internet.
- **Private Database Connection**:
  - Peering connection set up between VPC and Cloud SQL via Service Networking connection, ensuring PostgreSQL database is not exposed to the internet.

### Filestore (NFS) Configuration
- **Filestore Instance**:
  - A Cloud Filestore instance (`*-nfs`) is provisioned in the VPC.
  - **Tier**: `BASIC_HDD` with a minimum capacity of `1TB` (1024 GB).
  - **File Share**: A share named `workspaces` is configured and mounted to `/workspaces` in the Cloud Run containers (requires Gen2 execution environment).
- **Internal DNS for NFS**:
  - A private DNS zone (`apf.internal.`) is established with an `A` record (`nfs.apf.internal.`) pointing to the private IP of the Filestore instance to allow static mounting configurations.
- **Firewall Rules**:
  - Ports `111`, `2049`, and `20048` (TCP & UDP) must be allowed internally within the VPC to permit NFS traffic from the subnet range.

### Security & IAM
- **Least-Privilege Service Accounts**:
  - **Ops Service Account (`*-ops`)**: Used by CI/CD toolchain (Cloud Build/Deploy). Holds role admin permissions on Run, Storage, Artifact Registry, Secret Manager, VPC Access, and Logging.
  - **Compute Service Account (`*-compute`)**: Used by Cloud Run containers at runtime. Granted narrow roles: `roles/datastore.user` (Firestore), `roles/storage.objectAdmin` (GCS), `roles/secretmanager.secretAccessor`, `roles/cloudtrace.agent`, `roles/cloudsql.client`.
- **Secret Manager**:
  - Storage of `db-password`, `github-pat`, `secrets-encryption-key`, and `iap-client-secret` with access restricted strictly to the runtime Compute service account.

### Compliance
- **Data-in-Transit Encryption**: All endpoints are secured using HTTPS (SSL terminated at Load Balancer via Google-managed SSL Certificate). Service-to-service communication within GCP network uses Google-managed TLS.
- **Data-at-Rest Encryption**:
  - GCS Buckets, Filestore instances, Firestore, and Cloud SQL databases are encrypted at rest by default.
  - Custom encryption key (`SECRETS_ENCRYPTION_KEY`) is used to double-encrypt sensitive application data before saving to Datastore.
- **Container Vulnerability Scanning**: Active scanning configured on the Artifact Registry Docker repositories.

### Monitoring & Logging
- **Cloud Monitoring Dashboards**:
  - **Log Monitor Dashboard**: Visualizes request rates, p99 latencies, and service logs for UI, API, and AI services.
  - **Trace Monitor Dashboard**: Visualizes service-to-service request latency heatmaps.
- **Cloud Trace**: Enabled across UI, API, and AI services for monitoring distributed tracing.
- **Cloud Audit Logs**: Enabled for GCP resources to trace modification of IAM permissions, secret access, and administrative changes.
