const core = require("@actions/core");
const github = require("@actions/github");
const fs = require("fs").promises;

const planJson = core.getInput("plan-json");
const planSummary = core.getInput("plan-summary");
const applySummary = core.getInput("apply-summary");
const ghToken = core.getInput("token");
const oktokit = github.getOctokit(ghToken);

async function createCheck() {
  const { context } = github;

  const plan = JSON.parse(await fs.readFile(planJson, "utf-8"));
  const humanSummary = await fs.readFile(planSummary, "utf-8");
  let applyLog;
  if (applySummary) {
    try {
      applyLog = await fs.readFile(applySummary, "utf-8");
    } catch {}
  }

  function countActions(plan, type) {
    return plan.resource_changes.filter((ch) =>
      ch.change.actions.includes(type)
    ).length;
  }
  const createCount = countActions(plan, "create");
  const updateCount = countActions(plan, "update");
  const deleteCount = countActions(plan, "delete");

  const noChanges = createCount == 0 && updateCount == 0 && deleteCount == 0;
  const title = noChanges
    ? "No changes"
    : context.eventName === "push"
    ? `${createCount} added, ${updateCount} changed, ${deleteCount} destroyed`
    : `${createCount} to add, ${updateCount} to change, ${deleteCount} to destroy`;

  const codefence = "```";
  const summary = `
# Terraform Plan
${codefence}
${humanSummary.trim()}
${codefence}
${
  !!applyLog
    ? `
# Terraform Apply
${codefence}
${applyLog.replace(/\u001b\[[^m]+m/g, "").trim()}
${codefence}
`
    : ""
}
`;

  const sha =
    context.eventName === "pull_request"
      ? context.payload.pull_request?.head.sha
      : context.sha;
  await oktokit.rest.checks.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    head_sha: sha,
    status: "completed",
    conclusion: noChanges ? "neutral" : "success",
    name: context.eventName === "push" ? "Apply" : "Plan",
    output: {
      title,
      summary,
    },
  });
}

createCheck();
