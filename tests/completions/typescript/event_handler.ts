interface FormData {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
}

interface ValidationErrors {
  username?: string;
  email?: string;
  password?: string;
  confirmPassword?: string;
}

class FormValidator {
  validateUsername(username: string): string | null {
    if (username.length < 3) {
      return 'Username must be at least 3 characters';
    }
    if (!/^[a-zA-Z0-9_]+$/.test(username)) {
      return 'Username can only contain letters, numbers, and underscores';
    }
    return null;
  }

  validateEmail(email: string): string | null {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      return 'Invalid email format';
    }
    return null;
  }

  validatePassword(password: string): string | null {
    if (password.length < 8) {
      return 'Password must be at least 8 characters';
    }
    if (!/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(password)) {
      return 'Password must contain uppercase, lowercase, and numbers';
    }
    return null;
  }

  validateForm(data: FormData): ValidationErrors {
    const errors: ValidationErrors = {};

    const usernameError = this.validateUsername(data.username);
    if (usernameError) {
      errors.username = usernameError;
    }

    const emailError = this.validateEmail(data.email);
    if (emailError) {
      errors.email = emailError;
    }

    const passwordError = this.validatePassword(data.password);
    if (passwordError) {
      errors.password = passwordError;
    }

    if (data.password !== data.confirmPassword) {
      errors.confirmPassword = <CURSOR>;
    }

    return errors;
  }
}

export class RegistrationForm {
  private validator: FormValidator;
  private formData: FormData;
  private errors: ValidationErrors;

  constructor() {
    this.validator = new FormValidator();
    this.formData = {
      username: '',
      email: '',
      password: '',
      confirmPassword: '',
    };
    this.errors = {};
  }

  handleSubmit(): boolean {
    this.errors = this.validator.validateForm(this.formData);
    return Object.keys(this.errors).length === 0;
  }
}

// Expected: completion should add error message for password mismatch
