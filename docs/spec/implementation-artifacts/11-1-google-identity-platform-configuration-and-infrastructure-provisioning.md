# Story 11.1: Google Identity Platform Configuration & Infrastructure Provisioning

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Systems Administrator / Platform Engineer,  
I want to provision and configure the Google Identity Platform using Infrastructure-as-Code (IaC),  
so that user authentication, email/password sign-in, Google IDP integration, and multi-tenant capabilities are enabled at the project level.

## Acceptance Criteria

1. **Given** the Google Cloud project environment is initialized and APIs are enabled
2. **When** the Terraform module runs and applies the configuration
3. **Then** the Google Identity Platform is configured for the project
4. **And** the Email/Password sign-in provider is enabled (`password_required = true`) with duplicate emails disallowed
5. **And** Multi-tenancy is enabled on the Identity Platform (`allow_tenants = true`)
6. **And** if Google OAuth client credentials are provided, the default supported IDP configuration is successfully created and enabled for `google.com` using the IAP Client ID and Client Secret sourced securely from Secret Manager

## Tasks / Subtasks

- [x] Configure `google_identity_platform_config` resource with multi-tenancy and email/password provider (AC: 3, 4, 5)
- [x] Integrate Google OAuth `google_identity_platform_default_supported_idp_config` with client ID and secret sourced from Secret Manager (AC: 6)
- [x] Configure Terraform google & google-beta providers with `user_project_override = true` and `billing_project` to avoid quota/billing errors (AC: 2)
- [x] Clean up and remove deprecated plaintext client credentials variables across modules, staging, and prod configurations (AC: 2)
- [x] Run `terraform plan` and `terraform apply` to verify deployment on Staging environment (AC: 2, 3, 4, 5, 6)
- [ ] Apply and verify Terraform configuration on Production environment (AC: 2)

## Dev Notes

### Architecture & Security Constraints
- **Billing and Project Override:** Google Identity Platform APIs require explicit billing association. Thus, the Terraform providers (`google` and `google-beta`) must have `user_project_override = true` and `billing_project = var.project_id` configured to avoid API quota errors during resource updates.
- **Secure Secret Sourcing:** Plaintext secrets are completely eliminated. The Google IDP client secret is retrieved dynamically using the Google Secret Manager data source: `data.google_secret_manager_secret_version.iap_secret.secret_data`.
- **Multi-Tenancy:** The Identity Platform configuration enables multi-tenancy (`allow_tenants = true`), paving the way for tenant-specific silos and customer organization boundaries.

### Source Tree Components to Touch
- [identity_platform.tf](infra/modules/identity_platform.tf) — Core Identity Platform configuration.
- [variables.tf (Modules)](infra/modules/variables.tf) — Variable cleanup.
- [main.tf (Staging)](infra/staging/main.tf) and [variables.tf (Staging)](infra/staging/variables.tf) — Provider overrides and variables cleanup for staging.
- [main.tf (Prod)](infra/prod/main.tf) and [variables.tf (Prod)](infra/prod/variables.tf) — Provider overrides and variables cleanup for production.

### References
- [identity_platform.tf](infra/modules/identity_platform.tf)
- [epics.md#Epic 11: Identity and Access Management](docs/spec/planning-artifacts/epics.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

N/A

### Completion Notes List

- Staging environment is successfully provisioned and applied using `terraform apply` (applied on 2026-06-19).
- Production environment configurations are updated in `infra/prod` but need to be planned and applied.

### File List

- `infra/modules/identity_platform.tf`
- `infra/modules/variables.tf`
- `infra/staging/main.tf`
- `infra/staging/variables.tf`
- `infra/prod/main.tf`
- `infra/prod/variables.tf`
