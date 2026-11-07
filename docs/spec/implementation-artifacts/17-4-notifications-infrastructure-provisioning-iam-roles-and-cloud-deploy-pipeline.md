# Story 17.4: Notifications Infrastructure Provisioning, IAM Roles & Cloud Deploy Pipeline

Status: ready-for-dev

## Story

As a DevOps and Infrastructure Engineer,
I want Terraform infrastructure definitions, IAM role bindings, and Cloud Deploy manifests for notification services,
so that the dedicated notification sender service account and environment configurations are securely provisioned and deployed across staging and production environments.

## User Request Context & Specifics

1. Provision dedicated notification sender Service Account in Terraform: `google_service_account.notifications_sender` (`${prefix}-notifications-sender`).
2. Bind `roles/iam.serviceAccountTokenCreator` on the sender SA to the Cloud Run compute SA (`${prefix}-compute`).
3. Add Terraform variables `notifications_project_id`, `notifications_campaign_id`, `notifications_service_endpoint`, `email_service_type` in `infra/modules/variables.tf`.
4. Export output `notifications_sender_service_account_email` in `infra/modules/outputs.tf`.
5. Inject `NOTIFICATIONS_*` environment parameters into Cloud Run API service container in `infra/cicd/manifests/api.yaml`.
6. Parameterize `apf-staging` and `apf-production` targets in `infra/cicd/clouddeploy.yaml`.
7. Update environment templates in `config.env.template` and `api/.env`.
8. Document CNP MDB group membership onboarding for `cnp-<project_id>-api-senders.prod`.

## Verified Design Architecture & Elicited Decisions

- **Dedicated Service Account**: Dedicated sender SA isolates email notification authority.
- **Compute Impersonation**: Compute SA generates short-lived OAuth2 access tokens for the sender SA via IAM Token Creator role.
- **Unified Parameter Naming**: Standardized on `NOTIFICATIONS_` prefix across environment variables, manifests, and Terraform variables.
- **CNP MDB Group**: Sender service account email is added as `child_user` in `cnp-<project_id>-api-senders.prod`.

## Acceptance Criteria

1. **Given** `infra/modules/service_accounts.tf`, **When** applying Terraform, **Then** it MUST provision a dedicated service account `google_service_account.notifications_sender` with `account_id = "${var.prefix}-notifications-sender"`.
2. **Given** `infra/modules/service_accounts.tf`, **When** configuring IAM permissions, **Then** it MUST bind `roles/iam.serviceAccountTokenCreator` on `google_service_account.notifications_sender` to `serviceAccount:${google_service_account.compute.email}`.
3. **Given** `infra/modules/variables.tf` and `infra/modules/outputs.tf`, **When** configuring module interfaces, **Then** it MUST declare variables `notifications_project_id`, `notifications_campaign_id`, `notifications_service_endpoint`, `email_service_type`, and export output `notifications_sender_service_account_email`.
4. **Given** `infra/cicd/manifests/api.yaml`, **When** deploying the API Cloud Run service, **Then** container environment variables MUST include:
   - `EMAIL_SERVICE_TYPE`: `placeholder-email-service-type` (`${email_service_type}`)
   - `NOTIFICATIONS_PROJECT_ID`: `placeholder-notifications-project-id` (`${notifications_project_id}`)
   - `NOTIFICATIONS_CAMPAIGN_ID`: `placeholder-notifications-campaign-id` (`${notifications_campaign_id}`)
   - `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`: `placeholder-notifications-sender-sa` (`${notifications_sender_service_account}`)
   - `NOTIFICATIONS_SERVICE_ENDPOINT`: `placeholder-notifications-endpoint` (`${notifications_service_endpoint}`)
   - `NOTIFICATIONS_ENVIRONMENT`: `placeholder-notifications-env` (`${notifications_environment}`)
5. **Given** `infra/cicd/clouddeploy.yaml`, **When** configuring delivery targets, **Then** `apf-staging` and `apf-production` targets MUST include deploy parameters for `email_service_type`, `notifications_project_id`, `notifications_campaign_id`, `notifications_sender_service_account`, `notifications_service_endpoint`, and `notifications_environment`.
6. **Given** environment templates in `config.env.template` and `api/.env`, **When** configuring local environments, **Then** template variables for `EMAIL_SERVICE_TYPE=CONSOLE`, `NOTIFICATIONS_PROJECT_ID`, `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`, `NOTIFICATIONS_SERVICE_ENDPOINT`, and `NOTIFICATIONS_ENVIRONMENT=local` MUST be present.
7. **Given** CNP MDB group configuration, **When** onboarding the project in CNP, **Then** documentation MUST specify adding `${prefix}-notifications-sender@${project_id}.iam.gserviceaccount.com` as `child_user` in `cnp-<notifications_project_id>-api-senders.prod`.

## Tasks / Subtasks

- [ ] Task 1: Service Account & IAM Terraform Definition (AC: #1, #2, #3)
  - [ ] Add `google_service_account.notifications_sender` and IAM token creator binding in `infra/modules/service_accounts.tf`.
  - [ ] Add variables in `infra/modules/variables.tf` and outputs in `infra/modules/outputs.tf`.
- [ ] Task 2: Cloud Deploy & Cloud Run Manifests (AC: #4, #5)
  - [ ] Add `NOTIFICATIONS_*` container env vars to `infra/cicd/manifests/api.yaml`.
  - [ ] Add deploy parameters to `apf-staging` and `apf-production` in `infra/cicd/clouddeploy.yaml`.
- [ ] Task 3: Local Environment Templates & Documentation (AC: #6, #7)
  - [ ] Update `config.env.template` and `api/.env`.
  - [ ] Add onboarding instructions for CNP `api-senders` group.
