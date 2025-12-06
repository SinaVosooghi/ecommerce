// k6 configuration file
export const options = {
    // Output results to multiple destinations
    ext: {
        loadimpact: {
            // Cloud configuration (optional)
            projectID: process.env.K6_CLOUD_PROJECT_ID,
            name: 'Cart Service Load Test',
        },
    },
    
    // Default thresholds
    thresholds: {
        // 95% of requests should complete within 200ms
        http_req_duration: ['p(95)<200'],
        // Less than 1% of requests should fail
        http_req_failed: ['rate<0.01'],
    },
    
    // Summary output
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

// Scenarios for different test types
export const scenarios = {
    // Baseline: Normal expected load
    baseline: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '2m', target: 100 },
            { duration: '5m', target: 100 },
            { duration: '2m', target: 0 },
        ],
        gracefulRampDown: '30s',
    },
    
    // Peak: Black Friday simulation
    peak: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '2m', target: 200 },
            { duration: '5m', target: 500 },
            { duration: '10m', target: 500 },
            { duration: '3m', target: 200 },
            { duration: '2m', target: 0 },
        ],
        gracefulRampDown: '1m',
    },
    
    // Stress: Find breaking point
    stress: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '2m', target: 100 },
            { duration: '5m', target: 200 },
            { duration: '5m', target: 400 },
            { duration: '5m', target: 600 },
            { duration: '5m', target: 800 },
            { duration: '5m', target: 1000 },
            { duration: '5m', target: 0 },
        ],
        gracefulRampDown: '2m',
    },
    
    // Soak: Long-running test for memory leaks
    soak: {
        executor: 'constant-vus',
        vus: 100,
        duration: '2h',
    },
    
    // Spike: Sudden traffic spike
    spike: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '10s', target: 100 },
            { duration: '1m', target: 100 },
            { duration: '10s', target: 1000 }, // Spike!
            { duration: '3m', target: 1000 },
            { duration: '10s', target: 100 },
            { duration: '3m', target: 100 },
            { duration: '10s', target: 0 },
        ],
    },
};
