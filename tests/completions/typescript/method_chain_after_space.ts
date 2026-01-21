class DataProcessor {
  private data: number[];

  constructor(data: number[]) {
    this.data = data;
  }

  filter(predicate: (n: number) => boolean): DataProcessor {
    this.data = this.data.filter(predicate);
    return this;
  }

  map(mapper: (n: number) => number): DataProcessor {
    this.data = this.data.map(mapper);
    return this;
  }

  sum(): number {
    return this.data.reduce((a, b) => a + b, 0);
  }
}

const processor = new DataProcessor([1, 2, 3, 4, 5]);
const result = processor.filter(n => <CURSOR>).map(n => n * 2).sum();

// Expected: completion should complete the filter predicate
