interface Person {
  name: string;
  age: number;
  <CURSOR>
}

// Expected: completion might add another property like "email?: string;" or a method
