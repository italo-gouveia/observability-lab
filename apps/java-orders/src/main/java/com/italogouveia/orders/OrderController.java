package com.italogouveia.orders;

import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestClient;

/**
 * Orders service — the middle of the chain. The OpenTelemetry Java agent
 * auto-instruments the incoming Spring MVC request and the outbound RestClient
 * call to python-pricing, continuing the trace started by go-demo.
 */
@RestController
public class OrderController {

    private static final Logger log = LoggerFactory.getLogger(OrderController.class);

    private final RestClient pricing;

    public OrderController(RestClient pricingClient) {
        this.pricing = pricingClient;
    }

    @GetMapping("/orders")
    public Map<String, Object> createOrder() {
        int sku = ThreadLocalRandom.current().nextInt(1, 10);
        Map<?, ?> price = pricing.get()
                .uri("/price?sku=SKU-{sku}", sku)
                .retrieve()
                .body(Map.class);

        String orderId = UUID.randomUUID().toString();
        log.info("order {} confirmed for sku SKU-{}", orderId, sku);
        return Map.of("orderId", orderId, "status", "CONFIRMED", "price", price);
    }

    @GetMapping("/healthz")
    public Map<String, String> health() {
        return Map.of("status", "ok");
    }
}
