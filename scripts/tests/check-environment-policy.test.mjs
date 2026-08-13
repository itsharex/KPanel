import assert from 'node:assert/strict';
import test from 'node:test';

import {
  checkEnvironment,
  loadPolicy,
  validatePolicy,
} from '../check-environment-policy.mjs';

const policy = loadPolicy();

test('environment policy is structurally valid', () => {
  assert.deepEqual(validatePolicy(policy), []);
});

test('arena-154 accepts browser and candidate validation', () => {
  assert.equal(checkEnvironment(policy, 'arena-154', 'browser-validation').role, 'validation');
  assert.equal(checkEnvironment(policy, '154', 'candidate-validation').name, 'arena-154');
});

test('prod-108 rejects every validation purpose', () => {
  for (const purpose of [
    'candidate-validation',
    'browser-validation',
    'performance-validation',
    'failure-injection',
    'staging-deploy',
  ]) {
    assert.throws(() => checkEnvironment(policy, 'prod-108', purpose), /does not allow/);
  }
});

test('prod-108 only accepts deployment and deployment safety checks', () => {
  assert.equal(checkEnvironment(policy, '108', 'production-deploy').name, 'prod-108');
  assert.equal(checkEnvironment(policy, 'prod-108', 'production-safety-check').role, 'production');
});

test('unregistered hosts fail closed', () => {
  assert.throws(
    () => checkEnvironment(policy, 'temporary-host', 'browser-validation'),
    /environment is not registered/,
  );
});

test('production role cannot be configured to allow validation', () => {
  const invalid = structuredClone(policy);
  invalid.environments['prod-108'].allowedPurposes.push('browser-validation');
  assert.ok(validatePolicy(invalid).some((failure) => failure.includes('production environments must not allow')));
});
