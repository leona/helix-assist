import { Request, Response } from 'express';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  timestamp: number;
}

interface CreateUserRequest {
  name: string;
  email: string;
  password: string;
}

interface UserDTO {
  id: string;
  name: string;
  email: string;
  createdAt: string;
}

class UserService {
  async createUser(data: CreateUserRequest): Promise<UserDTO> {
    // Simulate user creation
    return {
      id: Math.random().toString(36),
      name: data.name,
      email: data.email,
      createdAt: new Date().toISOString(),
    };
  }

  async validateEmail(email: string): Promise<boolean> {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  }
}

const userService = new UserService();

export async function createUserHandler(req: Request, res: Response) {
  try {
    const { name, email, password } = req.body;

    if (!name || !email || !password) {
      return res.status(400).json({<CURSOR>});
    }

    const isValidEmail = await userService.validateEmail(email);
    if (!isValidEmail) {
      return res.status(400).json({
        success: false,
        error: 'Invalid email format',
        timestamp: Date.now(),
      });
    }

    const user = await userService.createUser({ name, email, password });

    return res.status(201).json({
      success: true,
      data: user,
      timestamp: Date.now(),
    });
  } catch (error) {
    return res.status(500).json({
      success: false,
      error: 'Internal server error',
      timestamp: Date.now(),
    });
  }
}

// Expected: completion should fill in the error response object
