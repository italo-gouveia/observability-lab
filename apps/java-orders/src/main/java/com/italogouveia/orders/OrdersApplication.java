package com.italogouveia.orders;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.client.RestClient;

@SpringBootApplication
public class OrdersApplication {

    public static void main(String[] args) {
        SpringApplication.run(OrdersApplication.class, args);
    }

    @org.springframework.context.annotation.Bean
    RestClient pricingClient(RestClient.Builder builder,
                             @org.springframework.beans.factory.annotation.Value("${pricing.url}") String pricingUrl) {
        return builder.baseUrl(pricingUrl).build();
    }
}
