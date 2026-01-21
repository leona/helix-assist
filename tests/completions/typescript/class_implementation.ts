interface Repository<T> {
  findById(id: string): Promise<T | null>;
  findAll(): Promise<T[]>;
  save(entity: T): Promise<T>;
  delete(id: string): Promise<boolean>;
}

interface User {
  id: string;
  name: string;
  email: string;
  createdAt: Date;
}

class UserRepository implements Repository<User> {
  private users: Map<string, User>;

  constructor() {
    this.users = new Map();
  }

  async findById(id: string): Promise<User | null> {
    return this.users.get(id) || null;
  }

  async findAll(): Promise<User[]> {
    return Array.from(this.users.values());
  }

  async save(entity: User): Promise<User> {
    this.users.set(entity.id, entity);
    return entity;
  }

  async delete(id: string): Promise<boolean> {
    <CURSOR>
  }

  async findByEmail(email: string): Promise<User | null> {
    for (const user of this.users.values()) {
      if (user.email === email) {
        return user;
      }
    }
    return null;
  }
}

// Expected: completion should implement the delete method body
