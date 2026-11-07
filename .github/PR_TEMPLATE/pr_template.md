# 🚀 Pull Request Overview

## 📖 Description & Motivation
<!-- 
Provide a clear, concise summary of what this PR accomplishes and why the change is necessary.
Include relevant background context, architectural decisions, or user pain points being addressed.
-->


## 🔗 Related Issues & Tickets
<!-- 
Link to the GitHub Issue, Jira ticket, or design spec that this PR addresses.
Example: `Fixes #58`, `Closes #59`, `Resolves APF-102`
-->
- Fixes #

## 🛠️ Type of Change
<!-- Select all relevant options by replacing `[ ]` with `[x]` -->
- [ ] 🐛 Bug Fix (non-breaking change which fixes an issue)
- [ ] ✨ New Feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking Change (fix or feature that causes existing functionality to fail)
- [ ] ♻️ Refactoring / Tech Debt (code improvements without feature changes)
- [ ] 📚 Documentation Update (README, CONTRIBUTING, architecture specs)
- [ ] ⚙️ Chore / Toolchain (Mise orchestration, CI/CD, dependency updates)

---

## 🔍 Proposed Technical Changes
<!--
Summarize the technical changes made, grouped logically by component/service layer.
Example:
### 🤖 Ai Service (`ai/`)
- Added new prompt template handling for workflow generation.

### 🌐 Ui Frontend (`ui/`)
- Refactored CanvasPane to support live streaming artifact content.
-->


---

## 🧪 Verification & Testing Plan

### 🤖 Automated Testing & Quality Gates
<!-- Confirm that all local quality gates pass successfully -->
- [ ] `mise run test` (Unit & Integration test suites pass)
- [ ] `mise run lint` (Code formatting and static analysis pass)
- [ ] `./scripts/test_mise_orchestration.sh` (Mise structural and port verification pass)

### 🖐️ Manual Verification & Browser Testing
<!-- Describe the exact manual testing steps taken to verify the changes -->
1. Ran `mise run local` to spin up the local stack.
2. Verified service availability on endpoints: `http://localhost:8080`, `http://localhost:8081`, `http://localhost:8082`.
3. [Add specific UI/API verification steps here]

---

## 📸 Screenshots & Recordings (UI / Terminal Logs)
<!--
If this PR includes visual UI changes or critical terminal output, attach before/after screenshots or logs here.
-->
| Before | After |
| :---: | :---: |
| <!-- Insert Before Image/Log --> | <!-- Insert After Image/Log --> |

---

## 🏁 Pre-Merge Checklist
<!-- Ensure all governance hygiene standards are met before requesting review -->
- [ ] 🧑‍💻 I have performed a thorough self-review of my own code.
- [ ] 📝 I have updated all relevant documentation (`README.md`, `CONTRIBUTING.md`, architecture specs).
- [ ] 🔒 I have verified that zero hardcoded secrets or API keys (e.g., `GEMINI_API_KEY`) are committed.
- [ ] 💬 I have added clear, descriptive comments to complex or non-obvious code sections.
- [ ] 🏷️ My commit messages follow Conventional Commits standards (e.g., `feat: ...`, `fix: ...`).
