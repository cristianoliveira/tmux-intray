import { DEFAULT_CONFIG, getEventConfig } from './config-loader.js';
console.log('DEFAULT_CONFIG keys:', Object.keys(DEFAULT_CONFIG));
console.log('DEFAULT_CONFIG.notifications keys:', Object.keys(DEFAULT_CONFIG.notifications));
console.log('session.idle:', DEFAULT_CONFIG.notifications['session.idle']);
const config = { ...DEFAULT_CONFIG };
console.log('config.notifications[\"session.idle\"]:', config.notifications['session.idle']);
const idleConfig = getEventConfig(config, 'session.idle');
console.log('idleConfig:', idleConfig);
// Also check if config.notifications['session.idle'] is mutated
config.notifications['session.idle'].message = 'Task done';
console.log('after mutation:', config.notifications['session.idle']);
