'use strict';

const fs = require('fs');
const path = require('path');

function readFileSafe(p) {
  try {
    return fs.readFileSync(p, 'utf-8');
  } catch {
    return null;
  }
}

function parseTopLevelScalars(text) {
  const out = {};
  for (const line of text.split(/\r?\n/)) {
    const m = line.match(/^([a-zA-Z0-9_]+):\s*(.+)$/);
    if (m) out[m[1]] = m[2].replace(/^["']|["']$/g, '');
  }
  return out;
}

function parseNestedBlock(text, parentKey) {
  const out = {};
  let inBlock = false;
  for (const line of text.split(/\r?\n/)) {
    if (line.match(new RegExp(`^${parentKey}:`))) {
      inBlock = true;
      continue;
    }
    if (!inBlock) continue;
    if (/^\S/.test(line) && !line.startsWith(' ')) break;
    const m = line.match(/^\s+([a-zA-Z0-9_]+):\s*(.+)$/);
    if (m) out[m[1]] = m[2].replace(/^["']|["']$/g, '');
  }
  return out;
}

/** Flat discover items: `  scope: done` under planning-status.yaml */
function parsePlanningStatusFlat(text) {
  const out = {};
  for (const line of text.split(/\r?\n/)) {
    const m = line.match(/^\s{2}([a-zA-Z0-9_]+):\s*(\S+)\s*$/);
    if (m) out[m[1]] = m[2];
  }
  return out;
}

function parseExecutionStatusMaps(text) {
  const epics = parseNestedBlock(text, 'epics');
  const stories = parseNestedBlock(text, 'stories');
  const legacy = parseNestedBlock(text, 'development_status');
  return { epics, stories, legacy };
}

function epicStatusFor(id, execMaps) {
  return execMaps.epics[id] || execMaps.legacy[id] || 'pending';
}

function nullIfLiteralNull(value) {
  if (value === undefined || value === null || value === 'null') return null;
  return value;
}

function parseEpicsFromReleasePlan(text) {
  const epics = [];
  const blocks = text.split(/\n\s*-\s+id:\s+/).slice(1);
  for (const block of blocks) {
    const id = block.match(/^(\S+)/)?.[1];
    const title = block.match(/title:\s*"?([^"\n]+)"?/)?.[1];
    const wsjf = parseFloat(block.match(/wsjf:\s*([\d.]+)/)?.[1] || '0');
    const file = block.match(/file:\s*(\S+)/)?.[1];
    if (id) epics.push({ id, title: title || id, wsjf, file });
  }
  return epics;
}

function parseSimpleEpic(text) {
  const title = text.match(/^title:\s*"?([^"\n]+)"?/m)?.[1];
  const stories = [];
  const storyBlocks = text.split(/\n\s*-\s+id:\s+/).slice(1);
  for (const sb of storyBlocks) {
    const sid = sb.match(/^(\S+)/)?.[1];
    const stitle = sb.match(/title:\s*"?([^"\n]+)"?/)?.[1];
    if (sid) stories.push({ id: sid, title: stitle || sid });
  }
  return { title, stories };
}

function readSpecsStatus(projectDir) {
  const specsDir = path.join(projectDir, 'specs');
  const stateText = readFileSafe(path.join(specsDir, 'state.yaml')) || '';
  const releaseText = readFileSafe(path.join(specsDir, 'release-plan.yaml')) || '';
  const execText = readFileSafe(path.join(specsDir, 'execution-status.yaml')) || '';
  const planningText = readFileSafe(path.join(specsDir, 'planning-status.yaml')) || '';

  const stateScalars = parseTopLevelScalars(stateText);
  const state = {
    active_flow: stateScalars.active_flow,
    active_epic_id: nullIfLiteralNull(
      stateScalars.active_epic_id || stateScalars.active_epic
    ),
    active_story_id: nullIfLiteralNull(stateScalars.active_story_id),
    active_bug_id: nullIfLiteralNull(stateScalars.active_bug_id),
    release: parseNestedBlock(stateText, 'release'),
    git: parseNestedBlock(stateText, 'git'),
    handoff: parseNestedBlock(stateText, 'handoff'),
    epic_cycle: parseNestedBlock(stateText, 'epic_cycle'),
  };

  const release = parseNestedBlock(releaseText, 'release');
  const execMaps = parseExecutionStatusMaps(execText);
  const epics = parseEpicsFromReleasePlan(releaseText);
  const epicsWithStatus = epics.map((e) => ({
    ...e,
    status: epicStatusFor(e.id, execMaps),
  }));

  const activeEpicId = state.active_epic_id;
  const nextEpicId =
    epicsWithStatus.find((e) => e.status !== 'done')?.id || null;
  let activeEpic = null;
  const epicMeta = epicsWithStatus.find((e) => e.id === activeEpicId);
  if (epicMeta && epicMeta.file) {
    const epicText = readFileSafe(path.join(specsDir, epicMeta.file));
    if (epicText) activeEpic = parseSimpleEpic(epicText);
  }

  const planning = planningText ? parsePlanningStatusFlat(planningText) : {};

  return {
    projectDir,
    state,
    release,
    epics: epicsWithStatus,
    execution_status: { ...execMaps.epics, ...execMaps.legacy },
    story_status: execMaps.stories,
    planning_status: planning,
    active_epic: activeEpic,
    active_epic_id: activeEpicId,
    next_epic_id: activeEpicId ? null : nextEpicId,
  };
}

module.exports = { readSpecsStatus };

if (require.main === module) {
  const dir = process.argv[2] || process.cwd();
  console.log(JSON.stringify(readSpecsStatus(path.resolve(dir)), null, 2));
}
