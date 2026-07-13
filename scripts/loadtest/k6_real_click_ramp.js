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

function parseStages(spec) {
	return spec.split(',').map((item) => {
		const [duration, target] = item.split(':');
		return { duration: duration.trim(), target: Number(target) };
	});
}

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

export const options = {
	scenarios: {
		real_click_ramp: {
			executor: 'ramping-arrival-rate',
			startRate: Number(__ENV.START_RATE || 5),
			timeUnit: '1s',
			preAllocatedVUs: Number(__ENV.PREALLOCATED_VUS || 40),
			maxVUs: Number(__ENV.MAX_VUS || 400),
			stages: parseStages(__ENV.STAGES || '1m:10,2m:25,2m:50,2m:75,1m:0'),
		},
	},
	summaryTrendStats: ['avg', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
	const ip = pickAllowedIP();
	const headers = browserHeaders(ip);

	const challengeRes = http.get(`${BASE_URL}/v1/challenge`, {
		headers,
		tags: { flow: 'challenge', scenario: 'real-click-ramp' },
	});

	const challengeOk = check(challengeRes, {
		'challenge endpoint returns 200': (res) => res.status === 200,
	});
	if (!challengeOk) {
		sleep(1);
		return;
	}

	const challenge = challengeRes.json();
	sleep(SOLVE_DELAY_MS / 1000);

	const payload = JSON.stringify({
		campaign_id: `k6-ramp-${exec.scenario.iterationInTest}`,
		challenge_id: challenge.challenge_id,
		challenge_token: computeChallengeToken(challenge.nonce),
	});

	const clickRes = http.post(`${BASE_URL}/click`, payload, {
		headers,
		tags: { flow: 'click', scenario: 'real-click-ramp' },
	});

	check(clickRes, {
		'real click stays successful': (res) => res.status === 200 && res.json('status') === 'success',
	});
}
