export const DEMO_DATA = {
  junctions: [
    {
      id: 0,
      label: "Junction 1",
      cycle: [
        {
          id: 0,
          signal_groups: [{ id: 0, signals: [
            { duration: 30, color: "GREEN" },
            { duration: 20, color: "RED" }
          ]}]
        },
        {
          id: 1,
          signal_groups: [{ id: 0, signals: [
            { duration: 20, color: "GREEN" },
            { duration: 15, color: "RED" }
          ]}]
        }
      ],
      offset: 0,
      point: { x: 0, y: 0 }
    },
    {
      id: 1,
      label: "Junction 2",
      cycle: [
        {
          id: 10,
          signal_groups: [{ id: 0, signals: [
            { duration: 20, color: "RED" },
            { duration: 35, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]}]
        },
        {
          id: 11,
          signal_groups: [{ id: 0, signals: [
            { duration: 10, color: "RED" },
            { duration: 10, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]}]
        }
      ],
      offset: 0,
      point: { x: 0, y: 200 }
    },
    {
      id: 2,
      label: "Junction 3",
      cycle: [
        {
          id: 20,
          signal_groups: [{ id: 0, signals: [
            { duration: 45, color: "RED" },
            { duration: 10, color: "GREEN" }
          ]}]
        },
        {
          id: 21,
          signal_groups: [{ id: 0, signals: [
            { duration: 7, color: "RED" },
            { duration: 18, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]}]
        }
      ],
      offset: 0,
      point: { x: 0, y: 450 }
    },
    {
      id: 3,
      label: "Junction 4",
      cycle: [
        {
          id: 20,
          signal_groups: [{ id: 0, signals: [
            { duration: 40, color: "RED" },
            { duration: 15, color: "GREEN" }
          ]}]
        },
        {
          id: 21,
          signal_groups: [{ id: 0, signals: [
            { duration: 10, color: "RED" },
            { duration: 20, color: "GREEN" }
          ]}]
        }
      ],
      offset: 0,
      point: { x: 0, y: 600 }
    }
  ],
  desiredSpeed: 40.0
};

