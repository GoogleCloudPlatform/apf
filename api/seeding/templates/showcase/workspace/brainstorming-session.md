# Brainstorming Session: Showcase Features & Tours

**Date**: 2026-03-15  
**Status**: In Progress  
**Participants**: Product Manager, AI Assistant  

---

## 1. Session Goals
- Brainstorm core requirements for the on-demand **Foundry Showcase** feature.
- Identify the target user persona friction points (empty dashboards, terminal dependencies).
- Design a high-fidelity, self-referential tour mapping.

## 2. Key Ideas & Features Under Discussion
1. **Interactive Node Canvas**: Ensure users can see node grids representing the design steps.
2. **Deterministic DB Hydration**: Use deterministic IDs to avoid race conditions.
3. **Reactive UI Tour**: An overlay walkthrough introducing the secrets modal and onboarding panels.
4. **Instant Reset Hook (Restore to Default)**: A settings option to cleanly rebuild the workspace.

## 3. Open Discussion Points (In Progress)
- [ ] Should the showcase project include some mock code files under an `Implementation` folder to show a complete build? (Currently deferred to post-Solutioning).
- [ ] What is the optimal number of pre-populated messages to show in the sidebar without overwhelming the user?
- [ ] How to handle user secrets (like GitHub PATs) in the showcase? (Decision: Disable VCS sync by default for the showcase project to ensure immediate execution).
