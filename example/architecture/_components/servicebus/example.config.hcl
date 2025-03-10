locals {

  servicebus = {
    sku_name = "Standard"
    topics = [
      {
        name                      = "content-model-updated",
        default_message_ttl       = "P14D",
        enable_batched_operations = true,
        enable_partitioning       = false,
        status                    = "Active"
        max_size_in_megabytes     = 1024,
        POC                       = "TBD",
        support_ordering          = true,
        subscriptions = [
          {
            name               = "EuContentMiddlewareACE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareBGE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareCOMED",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareDPL",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePECO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePEPCO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareViktorLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareRoystonLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          }
        ]
      },
      {
        name                      = "content-entry-updated",
        default_message_ttl       = "P14D",
        enable_batched_operations = true,
        enable_partitioning       = false,
        status                    = "Active"
        max_size_in_megabytes     = 1024,
        POC                       = "TBD",
        support_ordering          = true,
        subscriptions = [
          {
            name               = "EuContentMiddlewareACE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareBGE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareCOMED",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareDPL",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePECO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePEPCO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareViktorLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareRoystonLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          }
        ]
      },
      {
        name                      = "content-cache-expired",
        default_message_ttl       = "P14D",
        enable_batched_operations = true,
        enable_partitioning       = false,
        status                    = "Active"
        max_size_in_megabytes     = 1024,
        POC                       = "TBD",
        support_ordering          = true,
        subscriptions = [
          {
            name               = "EuContentMiddlewareACE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareBGE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareCOMED",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareDPL",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePECO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePEPCO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareViktorLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareRoystonLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          }
        ]
      }
    ]
  }

}
