* How to run:
```bash
go run ./cmd/greenwave/main.go --conf ./cmd/greenwave/conf.toml
```

* How Swagger documentation has been prepared:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/greenwave/main.go --output ./app/rest/docs --outputTypes json
```

* JSON request example for route `/api/greenwave/extract`:
```json
{
  "desired_speed_kmh": 40.0,
  "junctions": [
    {
      "id": 0,
      "label": "",
      "cycle": [
        {
          "id": 0,
          "signals": [
            {
              "duration": 30,
              "color": "GREEN"
            },
            {
              "duration": 20,
              "color": "RED"
            }
          ]
        },
        {
          "id": 1,
          "signals": [
            {
              "duration": 20,
              "color": "GREEN"
            },
            {
              "duration": 15,
              "color": "RED"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 0
      }
    },
    {
      "id": 1,
      "label": "",
      "cycle": [
        {
          "id": 10,
          "signals": [
            {
              "duration": 20,
              "color": "RED"
            },
            {
              "duration": 35,
              "color": "GREEN"
            },
            {
              "duration": 5,
              "color": "YELLOW"
            }
          ]
        },
        {
          "id": 11,
          "signals": [
            {
              "duration": 10,
              "color": "RED"
            },
            {
              "duration": 10,
              "color": "GREEN"
            },
            {
              "duration": 5,
              "color": "YELLOW"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 200
      }
    },
    {
      "id": 2,
      "label": "",
      "cycle": [
        {
          "id": 20,
          "signals": [
            {
              "duration": 45,
              "color": "RED"
            },
            {
              "duration": 10,
              "color": "GREEN"
            }
          ]
        },
        {
          "id": 21,
          "signals": [
            {
              "duration": 7,
              "color": "RED"
            },
            {
              "duration": 18,
              "color": "GREEN"
            },
            {
              "duration": 5,
              "color": "YELLOW"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 450
      }
    },
    {
      "id": 3,
      "label": "",
      "cycle": [
        {
          "id": 20,
          "signals": [
            {
              "duration": 40,
              "color": "RED"
            },
            {
              "duration": 15,
              "color": "GREEN"
            }
          ]
        },
        {
          "id": 21,
          "signals": [
            {
              "duration": 10,
              "color": "RED"
            },
            {
              "duration": 20,
              "color": "GREEN"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 600
      }
    }
  ]
}
```

* JSON response example for route `/api/greenwave/extract`:
```json
{
  "green_waves": [
    [
      {
        "interval_jun_one": {
          "phase_idx": 0,
          "start": 2,
          "end": 30
        },
        "interval_jun_two": {
          "phase_idx": 0,
          "start": 20,
          "end": 48
        },
        "distance": 200,
        "travel_time": 18,
        "band_width": 28
      },
      {
        "interval_jun_one": {
          "phase_idx": 1,
          "start": 52,
          "end": 62
        },
        "interval_jun_two": {
          "phase_idx": 1,
          "start": 70,
          "end": 80
        },
        "distance": 200,
        "travel_time": 18,
        "band_width": 10
      }
    ],
    [
      {
        "interval_jun_one": {
          "phase_idx": 0,
          "start": 22.5,
          "end": 32.5
        },
        "interval_jun_two": {
          "phase_idx": 0,
          "start": 45,
          "end": 55
        },
        "distance": 250,
        "travel_time": 22.5,
        "band_width": 10
      },
      {
        "interval_jun_one": {
          "phase_idx": 0,
          "start": 39.5,
          "end": 55
        },
        "interval_jun_two": {
          "phase_idx": 1,
          "start": 62,
          "end": 77.5
        },
        "distance": 250,
        "travel_time": 22.5,
        "band_width": 15.5
      }
    ],
    [
      {
        "interval_jun_one": {
          "phase_idx": 0,
          "start": 51.5,
          "end": 55
        },
        "interval_jun_two": {
          "phase_idx": 1,
          "start": 65,
          "end": 68.5
        },
        "distance": 150,
        "travel_time": 13.5,
        "band_width": 3.5
      },
      {
        "interval_jun_one": {
          "phase_idx": 1,
          "start": 62,
          "end": 71.5
        },
        "interval_jun_two": {
          "phase_idx": 1,
          "start": 75.5,
          "end": 85
        },
        "distance": 150,
        "travel_time": 13.5,
        "band_width": 9.5
      }
    ]
  ],
  "through_green_waves": [
    {
      "intervals": [
        {
          "phase_idx": 0,
          "start": 11,
          "end": 14.5
        },
        {
          "phase_idx": 0,
          "start": 29,
          "end": 32.5
        },
        {
          "phase_idx": 0,
          "start": 51.5,
          "end": 55
        },
        {
          "phase_idx": 1,
          "start": 65,
          "end": 68.5
        }
      ],
      "depth": 4,
      "bandwidth": 3.5
    },
    {
      "intervals": [
        {
          "phase_idx": 0,
          "start": 21.5,
          "end": 30
        },
        {
          "phase_idx": 0,
          "start": 39.5,
          "end": 48
        },
        {
          "phase_idx": 1,
          "start": 62,
          "end": 70.5
        },
        {
          "phase_idx": 1,
          "start": 75.5,
          "end": 84
        }
      ],
      "depth": 4,
      "bandwidth": 8.5
    }
  ]
}
```

* JSON request example for route `/api/greenwave/optimize`:
```json
{
  "optimizer_type": "genetic",
  "optimizer_params": {
    "population_size": 50,
    "generations": 100,
    "mutation_rate": 0.1,
    "tournament_size": 3,
    "crossover_type": "blend"
  },
  "desired_speed_kmh": 40.0,
  "junctions": [
    {
      "id": 0,
      "label": "",
      "cycle": [
        {
          "id": 0,
          "signals": [
            {
              "duration": 30,
              "color": "GREEN"
            },
            {
              "duration": 20,
              "color": "RED"
            }
          ]
        },
        {
          "id": 1,
          "signals": [
            {
              "duration": 20,
              "color": "GREEN"
            },
            {
              "duration": 15,
              "color": "RED"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 0
      }
    },
    {
      "id": 1,
      "label": "",
      "cycle": [
        {
          "id": 10,
          "signals": [
            {
              "duration": 20,
              "color": "RED"
            },
            {
              "duration": 35,
              "color": "GREEN"
            },
            {
              "duration": 5,
              "color": "YELLOW"
            }
          ]
        },
        {
          "id": 11,
          "signals": [
            {
              "duration": 10,
              "color": "RED"
            },
            {
              "duration": 10,
              "color": "GREEN"
            },
            {
              "duration": 5,
              "color": "YELLOW"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 200
      }
    },
    {
      "id": 2,
      "label": "",
      "cycle": [
        {
          "id": 20,
          "signals": [
            {
              "duration": 45,
              "color": "RED"
            },
            {
              "duration": 10,
              "color": "GREEN"
            }
          ]
        },
        {
          "id": 21,
          "signals": [
            {
              "duration": 7,
              "color": "RED"
            },
            {
              "duration": 18,
              "color": "GREEN"
            },
            {
              "duration": 5,
              "color": "YELLOW"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 450
      }
    },
    {
      "id": 3,
      "label": "",
      "cycle": [
        {
          "id": 20,
          "signals": [
            {
              "duration": 40,
              "color": "RED"
            },
            {
              "duration": 15,
              "color": "GREEN"
            }
          ]
        },
        {
          "id": 21,
          "signals": [
            {
              "duration": 10,
              "color": "RED"
            },
            {
              "duration": 20,
              "color": "GREEN"
            }
          ]
        }
      ],
      "offset": 0,
      "point": {
        "x": 0,
        "y": 600
      }
    }
  ]
}
```

* JSON response example for route `/api/greenwave/optimize`:
```json
```