import { DEFAULT_CONFIG, getEventConfig } from './config-loader.js';
console.log('DEFAULT_CONFIG.notifications[\"session.idle\"]:', DEFAULT_CONFIG.notifications['session.idle']);
const config = { ...DEFAULT_CONFIG };
console.log('config.notifications[\"session.idle\"]:', config.notifications['session.idle']);
const idleConfig = getEventConfig(config, 'session.idle');
console.log('idleConfig:', idleConfig);
