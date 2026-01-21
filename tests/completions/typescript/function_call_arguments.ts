function processUser(name: string, age: number, email: string) {
  console.log(`User: ${name}, Age: ${age}, Email: ${email}`);
}

// Cursor in middle of line after opening parenthesis
processUser(<CURSOR>);

// Expected: completion should suggest function parameters
