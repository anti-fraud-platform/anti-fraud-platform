import http from 'k6/http';
import crypto from 'k6/crypto';
import { check, sleep } from 'k6';
import exec from 'k6/execution';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:9090').replace(/\/$/, '');
const CHALLENGE_SALT = 'af-js-check-v1';
const SOLVE_DELAY_MS = Number(__ENV.SOLVE_DELAY_MS || 250);
const ALLOWED_IPS = (__ENV.ALLOWED_IPS || '8.8.8.8,8.8.4.4,9.9.9.9,208.67.222.222')
	.split(',')
	.map((ip) => ip.trim())
	.filter(Boolean);
const RATE_LIMIT_IP = __ENV.RATE_LIMIT_IP || '8.8.8.8';
const GEO_BLOCKED_IP = __ENV.GEO_BLOCKED_IP || '1.1.1.1';
const DURATION = __ENV.DURATION || '2m';

function browserHeaders(ip) {
	return {
		'Content-Type': 'application/json',
		'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36',
		'Accept': '*/*',
		'Accept-Language': 'en-US,en;q=0.9',
		'Accept-Encoding': 'gzip, deflate, br',
		'Sec-Fetch-Site': 'same-origin',
		'Sec-Fetch-Mode': 'cors',
		'Sec-Ch-Ua': '"Chromium";v="126", "Not.A/Brand";v="99", "Google Chrome";v="126"',
		'X-Forwarded-For': ip,
	};
}

function pickAllowedIP() {
	const index = exec.scenario.iterationInTest % ALLOWED_IPS.length;
	return ALLOWED_IPS[index];
}

function computeChallengeToken(nonce) {
	return crypto.sha256(`${nonce}:${CHALLENGE_SALT}`, 'hex');
}

function sendSolvedClick(ip, scenarioTag) {
	const headers = browserHeaders(ip);
	const challengeRes = http.get(`${BASE_URL}/v1/challenge`, {
		headers,
		tags: { flow: 'challenge', scenario: scenarioTag },
	});

	if (challengeRes.status !== 200) {
		check(challengeRes, {
			'challenge stays available': (res) => res.status === 200,
		});
		return null;
	}

	const challenge = challengeRes.json();
	sleep(SOLVE_DELAY_MS / 1000);

	const payload = JSON.stringify({
		campaign_id: `${scenarioTag}-${exec.scenario.iterationInTest}`,
		challenge_id: challenge.challenge_id,
		challenge_token: computeChallengeToken(challenge.nonce),
	});

	return http.post(`${BASE_URL}/click`, payload, {
		headers,
		tags: { flow: 'click', scenario: scenarioTag },
	});
}

export const options = {
	scenarios: {
		allowed_success: {
			executor: 'constant-arrival-rate',
			rate: 8,
			timeUnit: '1s',
			duration: DURATION,
			preAllocatedVUs: 20,
			exec: 'allowedSuccess',
		},
		rate_limited: {
			executor: 'constant-arrival-rate',
			rate: 20,
			timeUnit: '1s',
			duration: DURATION,
			preAllocatedVUs: 30,
			exec: 'rateLimited',
		},
		geoip_blocked: {
			executor: 'constant-arrival-rate',
			rate: 4,
			timeUnit: '1s',
			duration: DURATION,
			preAllocatedVUs: 10,
			exec: 'geoBlocked',
		},
	},
	summaryTrendStats: ['avg', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function allowedSuccess() {
	const response = sendSolvedClick(pickAllowedIP(), 'allowed-success');
	if (!response) {
		return;
	}

	check(response, {
		'allowed click returns 200 success': (res) => res.status === 200 && res.json('status') === 'success',
	});
}

export function rateLimited() {
	const response = sendSolvedClick(RATE_LIMIT_IP, 'rate-limited');
	if (!response) {
		return;
	}

	check(response, {
		'rate-limited traffic returns 200 or 429': (res) => res.status === 200 || res.status === 429,
	});
}

export function geoBlocked() {
	const response = http.post(
		`${BASE_URL}/click`,
		JSON.stringify({ campaign_id: `geo-blocked-${exec.scenario.iterationInTest}` }),
		{
			headers: browserHeaders(GEO_BLOCKED_IP),
			tags: { flow: 'click', scenario: 'geoip-blocked' },
		},
	);

	check(response, {
		'geoip blocked traffic returns 403': (res) => res.status === 403,
	});
}
