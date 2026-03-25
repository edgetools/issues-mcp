---
area: Combat/Aggro
depends_on: []
id: COMBAT-AGGRO-001
source: vault/design/mechanics/aggro.md
spec_approved: false
status: backlog
title: Basic aggro generation from damage
---

## Problem Statement

When a player character deals damage to a mob, that mob should prioritize
attacking the damage dealer based on cumulative threat generated.

## Hypothesis

A numeric threat table per mob, incremented by damage dealt.

<!-- work-log -->

### Work Log

**2026-03-24T14:32:00Z | impl-agent**
Initial investigation complete.
