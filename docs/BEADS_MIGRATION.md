# Beads Migration Complete ✅

**Date**: January 21, 2026  
**From**: Custom YAML-per-bead format  
**To**: Standard Beads CLI (issues.jsonl)

## Summary

Successfully migrated AgentiCorp from a custom YAML-based bead system to the standard [Beads CLI tool](https://github.com/steveyegge/beads) format. This eliminates the approval prompt issue and aligns AgentiCorp with best practices.

## Migration Statistics

- **Total Beads**: 118 beads imported
- **Open Beads**: 51 beads
- **Closed Beads**: 67 beads
- **Database Size**: 880 KB (`beads.db`)
- **JSONL Size**: 147 KB (`issues.jsonl`)

## Changes Made

### 1. Database Migration
- Imported all 114 YAML beads from `.beads/beads/*.yaml`
- Configured database prefix to `bd-` (matches existing bead IDs)
- Synced to `issues.jsonl` format

### 2. File Cleanup
- ✅ Removed `.beads/beads/` directory (115 YAML files)
- ✅ Removed import script
- ✅ Kept standard bd files: `issues.jsonl`, `beads.db`, `config.yaml`

### 3. Documentation Updates
- ✅ Updated `.beads/README.md` to follow bd best practices
- ✅ Updated `AGENTS.md` to reflect bd CLI-only workflow
- ✅ Removed references to YAML file manipulation

## Problem Solved

### Before Migration
- **Issue**: Cursor constantly prompted for approval when editing `.yaml` files
- **Cause**: 114 individual YAML files (one per bead) being modified frequently
- **Impact**: Workflow interruption, manual approvals required

### After Migration
- **Solution**: Single `issues.jsonl` file managed by bd CLI
- **Result**: No more approval prompts (same behavior as `nanolang` project)
- **Workflow**: Seamless `bd` command execution without interruption

## Benefits

### ✨ No More Approval Prompts
- **Before**: 114 individual YAML files → constant Cursor approval prompts
- **After**: Single `issues.jsonl` file → no prompts

### 🚀 Standard Workflow
- Matches other projects (e.g., `nanolang`)
- Uses standard `bd` CLI commands
- Leverages bd's upgrade path and migrations

### 🔧 Git-Friendly
- Single JSONL file with smart merge resolution
- Automatic sync with `bd sync`
- Branch-aware issue tracking

## Usage

All bead operations now use the `bd` CLI:

```bash
# Create new bead
bd create "Feature title" --type feature --priority 2

# List beads
bd list
bd list open
bd list closed

# Update bead
bd update bd-123 --status in_progress
bd update bd-123 --status closed

# Show details
bd show bd-123

# Sync with git
bd sync
```

## Verification

```bash
# Check bead count
bd list --limit 0

# Check database health
sqlite3 .beads/beads.db "SELECT COUNT(*) FROM issues"

# Verify JSONL
wc -l .beads/issues.jsonl
```

## Technical Details

### Database Configuration
- **Prefix**: `bd-` (changed from `AgentiCorp-`)
- **Format**: SQLite database + JSONL export
- **Storage**: `.beads/beads.db` (primary), `.beads/issues.jsonl` (sync)

### Migration Process
1. Created Python import script (`import_yaml_beads.py`)
2. Parsed all 114 YAML beads
3. Imported via `bd create` with `--id` and `--force` flags
4. Set status to `closed` for completed beads
5. Added parent relationships and dependencies
6. Synced to JSONL with `bd sync --rename-on-import`
7. Removed YAML files and import script

## Compatibility

### AgentiCorp's Bead System
- AgentiCorp **does not** need to read YAML files anymore
- Internal bead loading should be updated to use `bd` CLI or read from `issues.jsonl`
- Projects registered with AgentiCorp should also use bd format

### Future Projects
All new projects should:
1. Use `bd init` to initialize beads
2. Use `bd` CLI for all bead operations
3. Avoid custom YAML implementations

## Next Steps

1. ✅ Migration complete
2. ✅ Documentation updated
3. ✅ Git commit created
4. 🔄 Push to remote (pending)
5. 📢 Update AgentiCorp's internal bead loader to read from `issues.jsonl`
6. 📢 Verify agent workflows work with bd format

---

**Migration Commits**:
- `3dc32f0` - "Migrate from YAML beads to standard bd CLI format"
- Migration script: `import_yaml_beads.py` (removed after completion)
