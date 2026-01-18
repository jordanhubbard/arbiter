# Public Relations Manager Persona

The Public Relations Manager persona serves as the professional, courteous interface between your development team and external contributors on GitHub. It monitors issues, responds promptly, creates internal tracking beads, and ensures timely communication throughout the issue lifecycle.

## Overview

This persona implements the requirements for an autonomous GitHub issue management agent that:

- ✅ Monitors GitHub repositories for open and stale issues
- ✅ Responds to new issues with polite, informative initial responses
- ✅ Creates beads for issues and links them to GitHub issues
- ✅ Calls issues to the attention of engineering and product manager personas
- ✅ Updates GitHub issues when corresponding beads have new comments or status changes
- ✅ Notifies contributors when releases containing fixes are published
- ✅ Ensures all team communication flows through the PR manager (not directly on GitHub)
- ✅ Acts as the polite and knowledgeable face of the project

## Configuration

### Prerequisites

1. **GitHub API Access**: The PR Manager needs GitHub API credentials to monitor and update issues
2. **Registered Projects**: Projects must be registered in Arbiter with GitHub repository URLs
3. **Bead System**: The `bd` (beads) CLI tool should be available for work tracking

### Setting Up GitHub Access

Configure GitHub credentials for the persona to use. Add to your Arbiter configuration:

```yaml
personas:
  public-relations-manager:
    enabled: true
    github:
      token: "${GITHUB_TOKEN}"  # Personal Access Token or GitHub App token
      # Required permissions: issues:read, issues:write, repo:read
    
    # Repository configuration
    repositories:
      - owner: "your-org"
        repo: "your-repo"
        labels:
          auto_respond: true  # Automatically respond to new issues
          stale_threshold_days: 30  # Mark issues stale after 30 days
      
      - owner: "your-org"
        repo: "another-repo"
        labels:
          auto_respond: true
          stale_threshold_days: 14

    # Response configuration
    response:
      acknowledgment_delay_hours: 24  # Respond within 24 hours
      update_frequency_hours: 1  # Check for updates every hour
      scan_frequency_minutes: 15  # Scan for new issues every 15 minutes

    # Priority mapping
    priority_mapping:
      bug: 2  # P2 - Medium
      security: 1  # P1 - High
      feature: 3  # P3 - Low
      question: 3  # P3 - Low
```

### Project Registration

Ensure your projects are registered with their GitHub repositories:

```yaml
projects:
  - id: my-project
    name: "My Project"
    git_repo: "https://github.com/your-org/your-repo"
    branch: main
    beads_path: .beads
    context:
      github:
        owner: "your-org"
        repo: "your-repo"
```

### GitHub Token Setup

Create a GitHub Personal Access Token with these permissions:
- `repo:status` - Read repository status
- `repo:public_repo` - Access public repositories (or `repo` for private)
- `issues:read` - Read issues
- `issues:write` - Comment on and update issues

Set the token as an environment variable:
```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

Or use a GitHub App for better rate limits and security.

## How It Works

### Continuous Monitoring

The PR Manager runs continuously and:

1. **Every 15 minutes**: Scans all registered GitHub repositories for new issues
2. **Every hour**: Checks for bead updates to reflect in GitHub issues
3. **Daily**: Reviews stale issues and reports to product manager
4. **On release**: Scans release notes and notifies affected contributors

### Issue Lifecycle

```
New GitHub Issue
    ↓
PR Manager detects → Posts acknowledgment comment
    ↓
Creates internal bead → Tags relevant personas (engineering, product, QA)
    ↓
Team works on bead → PR Manager monitors bead for changes
    ↓
Bead updated → PR Manager posts update to GitHub issue
    ↓
Fix merged to release → PR Manager detects and notifies contributor
    ↓
Bead closed → PR Manager closes GitHub issue with thank you
```

### Bead Creation and Linking

When a new issue is detected:

1. **Create Bead**: `bd create "Bug: [Issue title]" -p [priority]`
2. **Link to GitHub**: Store mapping in bead metadata: `github-issue: owner/repo#123`
3. **Tag Personas**: Notify appropriate team members based on issue type
4. **Track Status**: Monitor bead for status changes and comments

### GitHub Updates

When bead status changes:

- **In Progress**: "Our team is working on this issue."
- **Blocked**: "This issue is blocked by [reason]. We'll update when resolved."
- **Fixed**: "A fix has been submitted in PR #456."
- **Closed**: "This issue has been resolved in release v1.2.3. Thank you!"

## Response Templates

The persona uses customizable templates for consistency:

### New Issue Acknowledgment
```
Thank you for reporting this issue! We've received your report and will 
investigate. We'll keep this issue updated with our progress and findings.
```

### Status Update
```
Update: Our team has investigated this issue and identified the root cause. 
A fix is currently in progress. We'll notify you when it's included in a release.
```

### Fixed in Release
```
Good news! This issue has been fixed and is included in release v1.2.3. 
Thank you for reporting this and helping us improve the project!
```

Customize these templates in your configuration to match your project's voice.

## Integration with Other Personas

### Engineering Personas
- PR Manager creates beads and tags engineering team
- Engineers work on beads, add comments with technical details
- PR Manager translates technical updates into user-friendly GitHub comments

### Product Manager
- Receives daily reports on issue trends and priorities
- Makes decisions on ambiguous issues via decision beads
- Reviews stale issues flagged by PR Manager

### QA Persona
- Verifies fixes before PR Manager closes issues
- Tags issues that need regression testing
- Confirms issue resolution before notifying contributors

### Decision Maker
- Resolves decision beads for ambiguous issues
- Determines priorities for feature requests
- Handles controversial feedback

## Best Practices

1. **Timely Responses**: Configure acknowledgment within 24 hours
2. **Regular Updates**: Update issues at least weekly, even if just to say "still working on it"
3. **Clear Communication**: Avoid technical jargon when talking to external contributors
4. **Set Expectations**: Be honest about timelines and complexity
5. **Thank Contributors**: Always acknowledge and thank people for their time
6. **Never Promise**: Don't commit to features or timelines without authorization
7. **Security First**: Escalate security issues immediately to P0
8. **Maintain Consistency**: Use templates but personalize with specific details

## Monitoring and Metrics

The PR Manager tracks:
- Average time to first response
- Average time to resolution
- Number of stale issues
- Issue-to-bead mapping coverage
- Release notification success rate

Access metrics via:
```bash
curl http://localhost:8080/api/v1/personas/public-relations-manager/metrics
```

## Troubleshooting

### GitHub API Rate Limits
- Use GitHub App for higher rate limits (5000/hour vs 60/hour)
- Configure caching to minimize API calls
- Adjust scan frequency if hitting limits

### Issues Not Detected
- Verify GitHub token has correct permissions
- Check repository is registered in configuration
- Review logs for API errors

### Beads Not Created
- Ensure `bd` CLI is in PATH
- Verify project beads directory is initialized
- Check file permissions on beads directory

### Updates Not Posting
- Verify GitHub token has `issues:write` permission
- Check for webhook conflicts if using GitHub Apps
- Review rate limit status

## Example Usage

### Spawn the PR Manager

Via API:
```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "pr-manager-1",
    "persona_name": "examples/public-relations-manager",
    "project_id": "my-project"
  }'
```

Via Web UI:
1. Navigate to http://localhost:8080
2. Click "Spawn New Agent"
3. Select "public-relations-manager" persona
4. Select your project
5. Click "Spawn Agent"

The agent will immediately begin monitoring configured repositories.

### View Active Issues

```bash
curl http://localhost:8080/api/v1/beads?type=task&tags=github-issue
```

### Check GitHub Sync Status

```bash
curl http://localhost:8080/api/v1/personas/public-relations-manager/status
```

## Customization

### Adjust Autonomy Level

Edit `PERSONA.md` to change decision-making authority:
- **More Autonomous**: Let it handle more issue types without human approval
- **Less Autonomous**: Require decision beads for more cases

### Custom Response Templates

Create a templates directory:
```
personas/examples/public-relations-manager/templates/
  ├── new_issue.md
  ├── status_update.md
  ├── fixed_in_release.md
  ├── request_info.md
  └── stale_followup.md
```

Reference in configuration:
```yaml
personas:
  public-relations-manager:
    templates_path: "personas/examples/public-relations-manager/templates"
```

### Priority Rules

Customize issue-to-bead priority mapping:
```yaml
priority_mapping:
  label:security: 0  # P0 - Critical, needs human
  label:bug: 2  # P2 - Medium
  label:feature: 3  # P3 - Low
  label:good-first-issue: 3  # P3 - Low
  label:breaking: 1  # P1 - High
```

## Security Considerations

- **Token Security**: Never commit GitHub tokens to configuration files
- **Permission Scoping**: Use minimum required GitHub permissions
- **Private Information**: Don't expose internal discussions in public issues
- **Sensitive Issues**: Automatically mark security issues as private/confidential
- **Rate Limiting**: Implement exponential backoff for API calls
- **Audit Logging**: Log all external communications for compliance

## Contributing

To improve this persona:
1. Submit issues with your use cases and edge cases
2. Share your custom templates and configurations
3. Report bugs in GitHub monitoring or bead creation
4. Suggest improvements to response quality

## License

This persona follows the same license as the Arbiter project.
