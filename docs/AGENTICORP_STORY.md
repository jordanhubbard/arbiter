# AgentiCorp: The Story of Becoming CEO of Your Virtual Company

> **New to AgentiCorp?** Start here. This is the story of Sarah, who went from exhausted solo developer to CEO of her own virtual software company—overnight.

## Prologue: The Discovery

Sarah stared at her laptop screen, frustrated. Another late night debugging a memory leak in her startup's trading platform. Her small team of three developers was stretched thin—there was a backlog of 47 issues, a UI redesign waiting, and the DevOps migration to Kubernetes that nobody had time for.

"What if I had a whole company working on this?" she thought wistfully.

That's when she discovered AgentiCorp.

## Chapter 1: Becoming CEO

The installation was simple: `docker compose up -d`. When Sarah opened http://localhost:8080, she saw something unexpected—not just another project management tool, but an org chart:

```
                        CEO (Sarah)
                            |
            ┌───────────────┼───────────────┐
            |               |               |
     Product Manager   Engineering Mgr   QA Engineer
            |               |               |
            |       ┌───────┴───────┐       |
            |       |               |       |
     Project Manager  Code Reviewers  DevOps Engineer
            |                               |
            |                               |
    Documentation Mgr              Housekeeping Bot
```

### The CEO Role (That's You, Sarah)

As CEO, Sarah discovered she had three key responsibilities:

1. **Set the Vision**: File high-level goals as P0 decision beads
2. **Make Critical Decisions**: Approve/reject major technical decisions when the team escalates
3. **Break Deadlocks**: When her virtual employees disagree, she has the final say

She wasn't expected to write code, review PRs, or write tests. That's what her team was for.

## Chapter 2: Registering Her First Project

Sarah clicked "Add Project" and filled in the form:

```yaml
name: TradingPlatform
repository: git@github.com:sarah-startup/trading-platform.git
branch: main
beads_path: .beads
context:
  build_command: "go build ./..."
  test_command: "go test ./..."
  lint_command: "golangci-lint run"
```

Within seconds, AgentiCorp:
- Generated an SSH key for the project
- Asked her to add it as a GitHub deploy key (with write access)
- Cloned the repository
- Discovered 47 existing issues in her `.beads/issues.jsonl` file
- Started analyzing the codebase

## Chapter 3: Her First Day as CEO

### Morning: The Vision (9:00 AM)

Sarah created her first bead:

```bash
bd create "Q1 Goal: Reduce Trading Platform Latency to <10ms p99" \
  --type=epic \
  --priority=0 \
  --description="Our users are complaining about slow trades.
  We need to get p99 latency under 10ms for order placement."
```

She marked it P0 (critical) because that's CEO-level strategy.

### The Product Manager Springs to Action (9:05 AM)

Sarah watched in the UI as **Ava (Product Manager)** autonomously:

1. **Analyzed the Epic**: Read Sarah's goal
2. **Created a PRD**: Filed a new bead titled "PRD: Trading Platform Performance Optimization"
3. **Broke Down the Work**: Created 5 sub-beads:
   - Investigate current latency bottlenecks (assigned to Engineering Manager)
   - Profile database queries (engineering)
   - Evaluate caching strategies (architecture)
   - Design performance monitoring (DevOps)
   - Plan load testing strategy (QA)

Sarah didn't write any of those. Ava did.

### The Engineering Manager Takes Over (9:15 AM)

**Marcus (Engineering Manager)** got assigned the investigation bead. He autonomously:

1. **Read the codebase** (used `read_file` action on key files)
2. **Searched for slow queries** (used `search_text` action for "SELECT *")
3. **Analyzed the architecture** (created a document in docs/CURRENT_ARCHITECTURE.md)
4. **Filed his findings** as a new bead: "SRD: Database Query Optimization Strategy"

In his SRD (System Requirements Document), Marcus wrote:
- Current p99 latency: 47ms (unacceptable)
- Bottleneck: Order matching query does full table scan
- Recommendation: Add composite index on (symbol, timestamp)
- Estimated improvement: 10ms p99

### Escalation to CEO (10:30 AM)

Marcus created a **decision bead** tagged `[CEO]`:

> **Decision Required**: Add database index will improve latency but requires 4-hour migration downtime during market hours.
>
> Option 1: Take downtime Tuesday 2am-6am (low trading volume)
> Option 2: Implement eventually-consistent read replica first (2 weeks)
>
> Engineering recommends Option 1. DevOps concerned about migration risk.

Sarah's phone buzzed—AgentiCorp sent her a notification. She reviewed the decision in the UI:

- ✓ Read Marcus's analysis
- ✓ Read DevOps's concerns
- ✓ Checked the SRD document

Sarah clicked **Approve** and added:
> "Go with Option 1. Plan rollback procedure. Schedule for Tuesday 2am."

### The Team Executes (11:00 AM - 2:00 PM)

With Sarah's decision, the team sprang into action:

**Marcus (Engineering Manager)**:
- Created task beads for the implementation
- Assigned `implement-index-migration` to a code implementation agent

**Implementation Agent**:
- Read the database schema (`read_file` action)
- Wrote migration SQL (`write_file` action)
- Wrote rollback script
- Added tests (`run_tests` action to verify)
- Committed code (`git_commit` action)
- Pushed branch (`git_push` action)
- Created PR (`create_pr` action to #847)

**Sophia (Code Reviewer)**:
- Got auto-assigned when PR #847 was created (webhook trigger)
- Reviewed the migration script
- Checked for SQL injection risks
- Verified rollback procedure
- Added inline comment: "Add timeout to migration in case it takes >4hrs"
- Requested changes

**Implementation Agent** (responding to review):
- Read Sophia's feedback
- Added timeout parameter
- Updated PR
- Pushed changes

**Sophia** (second review):
- Verified timeout was added correctly
- Approved PR ✅

**Quinn (QA Engineer)**:
- Got automatically assigned a testing bead
- Wrote load tests (`write_file` action for test script)
- Ran tests against staging (`run_tests` action)
- Verified p99 latency: **8.7ms** ✅
- Filed test report bead with results

**Ryan (DevOps Engineer)**:
- Created deployment runbook
- Scheduled maintenance window
- Prepared monitoring alerts
- Updated status page

### Afternoon: Sarah Watches Her Company Work (3:00 PM)

Sarah clicked the "Project Viewer" dashboard. The activity feed showed:

```
✓ PR #847 merged by sophia (code-reviewer)
✓ Bead ac-x8q9 (implement-index-migration) closed by agent-1234
✓ Bead ac-p2r1 (load-testing) closed by quinn (qa-engineer)
✓ 5 beads in progress
✓ 2 beads blocked (waiting for deployment)
✓ Next deployment: Tuesday 2:00 AM
```

Sarah hadn't written a single line of code. Her virtual company did it all.

## Chapter 4: Understanding the Org Chart

### The C-Suite (Strategic Layer)

**CEO (You, Sarah)**
- Sets vision and priorities
- Makes final decisions on escalations
- Breaks deadlocks between departments
- NOT expected to code, review, or test

**Product Manager (Ava)**
- Identifies feature gaps
- Writes PRDs (Product Requirements Documents)
- Creates epics from CEO goals
- Prioritizes features
- Autonomy: High (can create beads, prioritize)

**Engineering Manager (Marcus)**
- Oversees technical feasibility
- Reviews architecture decisions
- Writes SRDs (System Requirements Documents)
- Creates implementation task breakdowns
- Autonomy: High (can assign work, create technical beads)

### The Management Layer

**Project Manager (Jamie)**
- Coordinates releases
- Tracks progress
- Manages schedules
- Enforces QA sign-off before release
- Autonomy: Semi-autonomous (can't skip QA)

**Code Reviewer (Sophia)**
- Reviews PRs for correctness and quality
- Checks security issues
- Verifies tests exist
- Approves/rejects PRs
- Autonomy: Full (can block releases)

**QA Engineer (Quinn)**
- Designs test strategies
- Writes and runs tests
- Verifies bug fixes
- Signs off on releases
- Autonomy: Full (can block releases)

**DevOps Engineer (Ryan)**
- Manages deployments
- Handles infrastructure
- Monitors production
- Writes runbooks
- Autonomy: High (can deploy to staging)

**Documentation Manager (Dana)**
- Keeps docs accurate
- Reviews doc changes in PRs
- Updates API documentation
- Maintains README
- Autonomy: Medium (follows doc policy)

### The Support Staff

**Housekeeping Bot**
- Cleans up stale beads
- Archives old issues
- Runs maintenance tasks
- Sends reminders
- Autonomy: Limited (routine tasks only)

## Chapter 5: The CEO's Daily Workflow

### Monday Morning: Strategy Session

Sarah reviews her inbox (AgentiCorp notifications):
- 3 decision beads require approval
- 2 P0 bugs filed automatically
- 1 release ready for approval

She spends 30 minutes making decisions, not writing code.

### Tuesday: Hands-Off Deployment

At 2:00 AM (while Sarah sleeps), AgentiCorp:
- Ran the migration
- Verified latency improvement
- Rolled back a config issue automatically
- Re-deployed successfully
- Sent Sarah a morning summary

Sarah woke up to: ✅ **Deployment successful. P99 latency now 8.2ms.**

### Wednesday: New Feature Request

Sarah types in the CEO chat interface:

> "Users want dark mode. How much effort?"

Within 5 minutes:
- Ava (Product Manager) filed a PRD bead
- Marcus (Engineering Manager) estimated 3 days
- Jamie (Project Manager) scheduled for Sprint 12

Sarah clicked **Approve** and that was it. The team started work.

## Chapter 6: How It All Works Behind the Scenes

### The Dispatch Loop (Every 10 Seconds)

```
┌─────────────────────────────────────┐
│   Dispatcher checks for ready work  │
├─────────────────────────────────────┤
│ 1. Find open beads                  │
│ 2. Match to idle agents by persona  │
│ 3. Load conversation history        │
│ 4. Send to LLM (Claude/GPT-4)      │
│ 5. Agent returns JSON actions       │
│ 6. Execute actions (read, write, commit) │
│ 7. Update bead with results         │
└─────────────────────────────────────┘
```

### The Agent Action System

Agents communicate through a structured JSON action schema:

```json
{
  "actions": [
    {"type": "read_file", "path": "src/auth.go"},
    {"type": "write_file", "path": "src/auth.go", "content": "..."},
    {"type": "run_tests", "test_pattern": "./..."},
    {"type": "git_commit", "commit_message": "feat: Add JWT auth"},
    {"type": "create_pr", "pr_title": "Add authentication"}
  ],
  "notes": "Implemented JWT authentication with tests"
}
```

Available actions include:
- **File operations**: read_file, write_file, edit_code, read_tree
- **Code quality**: run_tests, run_linter, build_project
- **Git operations**: git_commit, git_push, create_pr
- **Workflow**: start_development, whats_next, proceed_to_phase
- **Beads**: create_bead, close_bead, escalate_ceo

### The Motivation System (Proactive Agents)

Agents aren't just reactive—they wake up when needed:

- **Project Manager**: Wakes up 7 days before deadline
- **QA Engineer**: Wakes up when PRs are created
- **DevOps**: Wakes up when deployments are scheduled
- **Housekeeping Bot**: Wakes up weekly to clean

### The Escalation System

When agents disagree or get stuck:

1. Create decision bead
2. Tag with `[CEO]` or `[EM]` or `[PM]`
3. Right person gets notified
4. Decision made
5. Work unblocked

### The Feedback Loop

AgentiCorp implements a tight feedback cycle:

```
Code → Build → Lint → Test → Review → Deploy
  ↑                                      ↓
  └──────────── Fix if failed ───────────┘
```

Agents automatically:
- Run tests after code changes
- Run linters to check style
- Build to verify compilation
- Iterate until all checks pass

## Chapter 7: Sarah's Success (3 Months Later)

After 3 months with AgentiCorp:

- **180 beads completed** (vs. 23 before)
- **8 major features shipped** (vs. 1 before)
- **Zero production incidents** (QA caught them all)
- **Sarah's coding time**: 5 hours/week (vs. 60 hours/week)
- **Sarah's strategic time**: 15 hours/week (thinking, deciding, planning)

Sarah's startup raised Series A. When investors asked "How'd you ship so fast?" she smiled:

> "I built a company that works while I sleep."

## Chapter 8: Real-World Applications

### Startup Founders
- Focus on product-market fit while agents handle implementation
- Scale development without hiring
- Ship features 10x faster

### Solo Developers
- Get code review from virtual reviewers
- Automated testing and quality checks
- Never work alone again

### Small Teams
- Augment your team with specialized agents
- Cover gaps (no DevOps? Agent's got it)
- Focus humans on creative work

### Open Source Maintainers
- Agents triage issues
- Automated PR reviews
- Documentation stays current

## Epilogue: Your Turn

### Step 1: Install AgentiCorp

```bash
git clone https://github.com/jordanhubbard/agenticorp
cd agenticorp
docker compose up -d
open http://localhost:8080
```

### Step 2: Become CEO

Click "Register Project" and add your repo. AgentiCorp will:
- Generate SSH keys
- Clone your repo
- Load existing issues (if you use beads)
- Start analyzing your code

### Step 3: Set Your First Goal

```bash
bd create "Q1 Goal: [Your Big Goal]" \
  --type=epic \
  --priority=0 \
  --description="[What you want to achieve]"
```

### Step 4: Watch Your Company Work

Open the Project Viewer. Watch as:
- Product Manager creates PRD
- Engineering Manager breaks down work
- Agents write code, tests, docs
- Code Reviewers review PRs
- QA tests everything
- DevOps deploys

### Step 5: Make Decisions (Not Code)

When you get a `[CEO]` notification:
- Read the context
- Make the decision
- Let your team execute

### Step 6: Learn More

Once you're comfortable with the basics:
- Read [Agent Actions Reference](WORKFLOW_ACTIONS.md) for technical details
- Read [Git Workflow](GIT_SECURITY_MODEL.md) for version control integration
- Read [Architecture](../ARCHITECTURE.md) for system design
- Join the community on GitHub Discussions

---

## Frequently Asked Questions

### "Is this real AI or just automation?"

Real AI. Each agent is powered by frontier LLMs (Claude Sonnet, GPT-4) making autonomous decisions. They read code, write code, review PRs, and make technical judgments—not following scripts.

### "Do I need to train the agents?"

No. Agents come pre-trained with software engineering knowledge. You just configure your project (build command, test command) and they figure out the rest.

### "Can agents break my production?"

No. Agents can only:
- Work on agent/* branches (not main/master)
- Create PRs (you approve merges)
- Deploy to staging (production requires human approval)

Security constraints are built-in.

### "What if agents make mistakes?"

They do! That's why there's:
- Code review agents (catch bugs)
- QA agents (catch regressions)
- Build/lint/test feedback loops (catch errors)
- Human CEO approval for critical decisions

The system has checks and balances, like a real company.

### "How much does it cost?"

AgentiCorp is open source. You pay for:
- LLM API calls (Claude/OpenAI)
- Your own infrastructure (Docker host)

Typical cost: $50-200/month for a small project, depending on activity level.

### "Can I use my own LLM?"

Yes! AgentiCorp supports:
- Anthropic Claude (Opus, Sonnet, Haiku)
- OpenAI (GPT-4, GPT-3.5)
- Custom endpoints (bring your own LLM)

Configure in your provider registry.

---

**Welcome to AgentiCorp. You're now CEO of your own virtual software company.**

You think. They build.

---

*This story is the official onboarding document for AgentiCorp. For technical documentation, see [README.md](../README.md) and [ARCHITECTURE.md](../ARCHITECTURE.md).*

*Last updated: 2026-02-05*
