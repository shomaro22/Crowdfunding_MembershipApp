export const rampUpStages = [
  { duration: '30s', target: 5 },
  { duration: '1m', target: 10 },
  { duration: '30s', target: 0 },
];

export const spikeStages = [
  { duration: '20s', target: 5 },
  { duration: '10s', target: 30 },
  { duration: '20s', target: 5 },
  { duration: '30s', target: 0 },
];

export const stressStages = [
  { duration: '1m', target: 10 },
  { duration: '2m', target: 25 },
  { duration: '1m', target: 40 },
  { duration: '1m', target: 0 },
];

export const soakStages = [
  { duration: '1m', target: 8 },
  { duration: '5m', target: 8 },
  { duration: '1m', target: 0 },
];

export const smokeStages = [
  { duration: '30s', target: 3 },
  { duration: '30s', target: 0 },
];
