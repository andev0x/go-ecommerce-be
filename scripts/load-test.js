import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
export const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 20 },   // Ramp up to 20 users
    { duration: '5m', target: 50 },   // Stay at 50 users
    { duration: '2m', target: 100 },  // Ramp to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],  // 95% of requests under 200ms
    http_req_failed: ['rate<0.1'],     // Error rate under 10%
    errors: ['rate<0.1'],              // Custom error rate under 10%
  },
};

const BASE_URL = 'http://localhost:8080/api/v1';

// Test data
const testProduct = {
  name: `Test Product ${Math.random()}`,
  description: 'Load test product',
  price: 99.99,
  category_id: '550e8400-e29b-41d4-a716-446655440000', // Default category
  stock: 100,
  sku: `TEST-${Math.random().toString(36).substr(2, 9)}`,
};

export default function () {
  // Test scenarios with different weights
  const scenario = Math.random();
  
  if (scenario < 0.6) {
    // 60% - List products (most common operation)
    testListProducts();
  } else if (scenario < 0.8) {
    // 20% - Search products
    testSearchProducts();
  } else if (scenario < 0.9) {
    // 10% - Get single product
    testGetProduct();
  } else {
    // 10% - Create/Update/Delete operations
    testCRUDOperations();
  }
  
  sleep(1);
}

function testListProducts() {
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  const response = http.get(`${BASE_URL}/products?limit=20&offset=0`, params);
  
  const success = check(response, {
    'list products status is 200': (r) => r.status === 200,
    'list products response time < 200ms': (r) => r.timings.duration < 200,
    'list products has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success && Array.isArray(body.data.products);
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

function testSearchProducts() {
  const searchTerms = ['wireless', 'smart', 'gaming', 'bluetooth', 'laptop'];
  const term = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  const response = http.get(`${BASE_URL}/products/search?q=${term}&limit=10`, params);
  
  const success = check(response, {
    'search products status is 200': (r) => r.status === 200,
    'search products response time < 300ms': (r) => r.timings.duration < 300,
    'search products has results': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success && Array.isArray(body.data.products);
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

function testGetProduct() {
  // First get a list of products to get a valid ID
  const listResponse = http.get(`${BASE_URL}/products?limit=1`);
  
  if (listResponse.status === 200) {
    try {
      const listBody = JSON.parse(listResponse.body);
      if (listBody.data.products.length > 0) {
        const productId = listBody.data.products[0].id;
        
        const response = http.get(`${BASE_URL}/products/${productId}`);
        
        const success = check(response, {
          'get product status is 200': (r) => r.status === 200,
          'get product response time < 100ms': (r) => r.timings.duration < 100,
          'get product has data': (r) => {
            try {
              const body = JSON.parse(r.body);
              return body.success && body.data.id === productId;
            } catch {
              return false;
            }
          },
        });
        
        errorRate.add(!success);
      }
    } catch (e) {
      errorRate.add(true);
    }
  }
}

function testCRUDOperations() {
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  // Create product
  const createResponse = http.post(
    `${BASE_URL}/products`,
    JSON.stringify(testProduct),
    params
  );
  
  const createSuccess = check(createResponse, {
    'create product status is 201': (r) => r.status === 201,
    'create product response time < 300ms': (r) => r.timings.duration < 300,
  });
  
  if (createSuccess && createResponse.status === 201) {
    try {
      const createBody = JSON.parse(createResponse.body);
      const productId = createBody.data.id;
      
      // Update product
      const updateData = {
        name: `Updated ${testProduct.name}`,
        price: 149.99,
      };
      
      const updateResponse = http.put(
        `${BASE_URL}/products/${productId}`,
        JSON.stringify(updateData),
        params
      );
      
      const updateSuccess = check(updateResponse, {
        'update product status is 200': (r) => r.status === 200,
        'update product response time < 200ms': (r) => r.timings.duration < 200,
      });
      
      // Delete product
      const deleteResponse = http.del(`${BASE_URL}/products/${productId}`, null, params);
      
      const deleteSuccess = check(deleteResponse, {
        'delete product status is 200': (r) => r.status === 200,
        'delete product response time < 200ms': (r) => r.timings.duration < 200,
      });
      
      errorRate.add(!(createSuccess && updateSuccess && deleteSuccess));
    } catch (e) {
      errorRate.add(true);
    }
  } else {
    errorRate.add(true);
  }
}

// Setup function - runs once before the test
export function setup() {
  console.log('Starting load test for Product Service');
  
  // Verify service is running
  const response = http.get(`${BASE_URL}/../health`);
  if (response.status !== 200) {
    throw new Error('Product service is not healthy');
  }
  
  console.log('Product service is healthy, starting load test...');
}

// Teardown function - runs once after the test
export function teardown() {
  console.log('Load test completed');
}