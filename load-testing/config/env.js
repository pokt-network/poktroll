// Environment configuration
export const ENV_CONFIG = {
    anvilBaseUrl: __ENV.ANVIL_BASE_URL || 'http://localhost:8547',
    nginxBaseUrl: __ENV.NGINX_BASE_URL || 'http://localhost:8548',
    AppGateServerAnvilUrl: __ENV.APP_GATE_SERVER_ANVIL_URL || 'http://localhost:42069/anvil',
};
