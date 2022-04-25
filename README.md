# osqgen
Osquery helper tool for Elasticsearch 


#### Generate fields for integration package

```
osqgen --schema "./schema/osquery/osquery_schema_5.0.1.json" fields > osquery.yml
```

#### Generate fields for readme for integration package

```
osqgen --schema "./schema/osquery/osquery_schema_5.0.1.json" readme > readme.txt
```

#### Generate ECS fields for integration package

```
osqgen --schema "./schema/ecs/fields.ecs_1.12.yml" ecs > ecs.yml
```

Currently this extracts out all the ```date``` and ```ip``` fields and writes them out in the integration package fields format.
For example:
```
- external: ecs
  name:  client.ip
- external: ecs
  name:  client.nat.ip
- external: ecs
  name:  code_signature.timestamp
- external: ecs
  name:  destination.ip
# ........
```

