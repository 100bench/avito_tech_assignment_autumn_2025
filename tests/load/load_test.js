import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const createTeamDuration = new Trend('create_team_duration');
const createPRDuration = new Trend('create_pr_duration');
const reassignDuration = new Trend('reassign_duration');
const deactivateDuration = new Trend('deactivate_duration');

export const options = {
  stages: [
    { duration: '30s', target: 2 },
    { duration: '1m', target: 5 },
    { duration: '30s', target: 5 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<300', 'p(99.9)<300'],
    'http_req_failed{name:CreateTeam}': ['rate<0.001'],
    'http_req_failed{name:CreatePR}': ['rate<0.001'],
    'http_req_failed{name:Reassign}': ['rate<0.1'], // 10% для случаев NO_CANDIDATE
    create_team_duration: ['p(95)<300'],
    create_pr_duration: ['p(95)<300'],
    reassign_duration: ['p(95)<300'],
    deactivate_duration: ['p(95)<100'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('Подготовка тестовых данных...');
  
  const teams = [];
  const prs = [];
  
  // Создаем 20 команд по 10 пользователей
  for (let t = 0; t < 20; t++) {
    const teamName = `load-team-${t}-${Date.now()}`;
    const members = [];
    
    for (let u = 0; u < 10; u++) {
      members.push({
        user_id: `user-${t}-${u}-${Date.now()}`,
        username: `User${t}-${u}`,
        is_active: true,
      });
    }
    
    const res = http.post(`${BASE_URL}/team/add`, JSON.stringify({
      team_name: teamName,
      members: members,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (res.status === 201) {
      teams.push({ team_name: teamName, users: members.map(m => m.user_id) });
    }
  }
  
  // Создаем 50 PR
  for (let i = 0; i < 50; i++) {
    const team = teams[i % teams.length];
    const prID = `load-pr-${i}-${Date.now()}`;
    
    const res = http.post(`${BASE_URL}/pullRequest/create`, JSON.stringify({
      pull_request_id: prID,
      pull_request_name: `Load Test PR ${i}`,
      author_id: team.users[0],
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (res.status === 201) {
      prs.push({ pr_id: prID, team: team.team_name });
    }
  }
  
  console.log(`Создано команд: ${teams.length}, PR: ${prs.length}`);
  return { teams, prs };
}

export default function (data) {
  if (!data || !data.teams || data.teams.length === 0) {
    return;
  }
  
  const team = data.teams[Math.floor(Math.random() * data.teams.length)];
  
  // 1. Создание команды
  testCreateTeam();
  sleep(0.1);
  
  // 2. Создание PR
  testCreatePR(team);
  sleep(0.1);
  
  // 3. Переназначение ревьювера
  if (data.prs && data.prs.length > 0) {
    const pr = data.prs[Math.floor(Math.random() * data.prs.length)];
    testReassign(pr.pr_id);
    sleep(0.1);
  }
  
  // 4. Массовая деактивация (только для некоторых итераций)
  if (Math.random() < 0.3) { // 30% итераций
    const activeUsers = team.users.slice(1, 4); // Пропускаем автора
    if (activeUsers.length > 0) {
      testDeactivate(team.team_name, activeUsers);
    }
  }
  
  sleep(0.1);
}

function testCreateTeam() {
  const teamName = `test-team-${Date.now()}-${Math.random()}`;
  const start = Date.now();
  
  const res = http.post(`${BASE_URL}/team/add`, JSON.stringify({
    team_name: teamName,
    members: [
      { user_id: `u1-${Date.now()}-${Math.random()}`, username: 'User1', is_active: true },
      { user_id: `u2-${Date.now()}-${Math.random()}`, username: 'User2', is_active: true },
    ],
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'CreateTeam' },
  });
  
  const duration = Date.now() - start;
  createTeamDuration.add(duration);
  
  check(res, {
    'create team status 201': (r) => r.status === 201,
    'create team duration < 300ms': () => duration < 300,
  });
}

function testCreatePR(team) {
  const prID = `test-pr-${Date.now()}-${Math.random()}`;
  const start = Date.now();
  
  const res = http.post(`${BASE_URL}/pullRequest/create`, JSON.stringify({
    pull_request_id: prID,
    pull_request_name: 'Test PR',
    author_id: team.users[0],
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'CreatePR' },
  });
  
  const duration = Date.now() - start;
  createPRDuration.add(duration);
  
  check(res, {
    'create PR status 201': (r) => r.status === 201,
    'create PR duration < 300ms': () => duration < 300,
  });
}

function testReassign(prID) {
  const start = Date.now();
  
  // Получаем PR
  const getPR = http.get(`${BASE_URL}/pullRequest/get?pull_request_id=${prID}`);
  
  if (getPR.status !== 200) {
    return;
  }
  
  let prData;
  try {
    prData = JSON.parse(getPR.body);
  } catch {
    return;
  }
  
  if (!prData.pr || !prData.pr.assigned_reviewers || prData.pr.assigned_reviewers.length === 0) {
    return;
  }
  
  const oldReviewer = prData.pr.assigned_reviewers[0];
  
  const res = http.post(`${BASE_URL}/pullRequest/reassign`, JSON.stringify({
    pull_request_id: prID,
    old_user_id: oldReviewer,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'Reassign' },
  });
  
  const duration = Date.now() - start;
  reassignDuration.add(duration);
  
  check(res, {
    'reassign status 200 or 409': (r) => r.status === 200 || r.status === 409,
    'reassign duration < 300ms': () => duration < 300,
  });
}

function testDeactivate(teamName, userIDs) {
  if (!userIDs || userIDs.length === 0) {
    return;
  }
  
  const start = Date.now();
  
  const res = http.post(`${BASE_URL}/team/deactivateMembers`, JSON.stringify({
    team_name: teamName,
    user_ids: userIDs,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'Deactivate' },
  });
  
  const duration = Date.now() - start;
  deactivateDuration.add(duration);
  
  check(res, {
    'deactivate duration < 100ms': () => duration < 100,
  });
}

export function teardown(data) {
  console.log('Нагрузочное тестирование завершено');
}