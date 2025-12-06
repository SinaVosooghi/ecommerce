// k6 load test scenario - baseline
import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const cartCreations = new Counter('cart_creations');
const itemAdditions = new Counter('item_additions');
const errorRate = new Rate('errors');
const cartGetDuration = new Trend('cart_get_duration');

// Test configuration
export const options = {
    stages: [
        { duration: '1m', target: 50 },   // Ramp up to 50 users
        { duration: '3m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 50 },   // Ramp down to 50
        { duration: '1m', target: 0 },    // Ramp down to 0
    ],
    thresholds: {
        http_req_duration: ['p(95)<200', 'p(99)<500'],
        http_req_failed: ['rate<0.01'],
        errors: ['rate<0.01'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Generate unique user ID for each VU
function getUserId() {
    return `user-${__VU}-${__ITER}`;
}

// Generate random product ID
function getProductId() {
    const products = ['prod-001', 'prod-002', 'prod-003', 'prod-004', 'prod-005'];
    return products[Math.floor(Math.random() * products.length)];
}

// Generate random quantity
function getQuantity() {
    return Math.floor(Math.random() * 5) + 1;
}

// Generate random price in cents
function getPrice() {
    return Math.floor(Math.random() * 10000) + 100;
}

export default function () {
    const userId = getUserId();
    const headers = {
        'Content-Type': 'application/json',
        'Idempotency-Key': `${userId}-${Date.now()}`,
    };

    group('Cart Operations', function () {
        // 1. Add item to cart
        group('Add Item', function () {
            const addPayload = JSON.stringify({
                product_id: getProductId(),
                quantity: getQuantity(),
                unit_price: getPrice(),
            });

            const addRes = http.post(
                `${BASE_URL}/v1/cart/${userId}/items`,
                addPayload,
                { headers }
            );

            const addSuccess = check(addRes, {
                'add item status is 201': (r) => r.status === 201,
                'add item response has cart id': (r) => {
                    const body = JSON.parse(r.body);
                    return body.id !== undefined;
                },
            });

            if (addSuccess) {
                itemAdditions.add(1);
            } else {
                errorRate.add(1);
            }
        });

        sleep(0.5);

        // 2. Get cart
        group('Get Cart', function () {
            const startTime = Date.now();
            const getRes = http.get(`${BASE_URL}/v1/cart/${userId}`);
            cartGetDuration.add(Date.now() - startTime);

            const getSuccess = check(getRes, {
                'get cart status is 200': (r) => r.status === 200,
                'get cart has items': (r) => {
                    const body = JSON.parse(r.body);
                    return body.items && body.items.length > 0;
                },
            });

            if (!getSuccess) {
                errorRate.add(1);
            }
        });

        sleep(0.5);

        // 3. Add another item (50% of the time)
        if (Math.random() > 0.5) {
            group('Add Second Item', function () {
                const addPayload = JSON.stringify({
                    product_id: getProductId(),
                    quantity: getQuantity(),
                    unit_price: getPrice(),
                });

                const addRes = http.post(
                    `${BASE_URL}/v1/cart/${userId}/items`,
                    addPayload,
                    { headers: { ...headers, 'Idempotency-Key': `${userId}-${Date.now()}-2` } }
                );

                check(addRes, {
                    'add second item status is 201': (r) => r.status === 201,
                });

                if (addRes.status === 201) {
                    itemAdditions.add(1);
                }
            });

            sleep(0.3);
        }

        // 4. Get cart again to verify
        group('Verify Cart', function () {
            const getRes = http.get(`${BASE_URL}/v1/cart/${userId}`);
            
            check(getRes, {
                'verify cart status is 200': (r) => r.status === 200,
                'cart total is calculated': (r) => {
                    const body = JSON.parse(r.body);
                    return body.total_price > 0;
                },
            });
        });

        sleep(1);
    });
}

// Lifecycle hooks
export function setup() {
    console.log('Starting baseline load test...');
    console.log(`Target URL: ${BASE_URL}`);
    
    // Verify service is healthy
    const healthRes = http.get(`${BASE_URL}/health`);
    if (healthRes.status !== 200) {
        throw new Error('Service is not healthy');
    }
    
    return { startTime: Date.now() };
}

export function teardown(data) {
    const duration = (Date.now() - data.startTime) / 1000;
    console.log(`Load test completed in ${duration} seconds`);
}
