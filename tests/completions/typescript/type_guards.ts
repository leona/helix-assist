type PaymentMethod =
  | { type: 'credit_card'; cardNumber: string; cvv: string; expiryDate: string }
  | { type: 'paypal'; email: string }
  | { type: 'bank_transfer'; accountNumber: string; routingNumber: string }
  | { type: 'crypto'; walletAddress: string; currency: string };

interface Order {
  id: string;
  items: Array<{ productId: string; quantity: number; price: number }>;
  total: number;
  paymentMethod: PaymentMethod;
  status: 'pending' | 'processing' | 'completed' | 'failed';
}

class PaymentProcessor {
  processPayment(order: Order): Promise<boolean> {
    const { paymentMethod } = order;

    switch (paymentMethod.type) {
      case 'credit_card':
        return this.processCreditCard(paymentMethod.cardNumber, order.total);
      case 'paypal':
        return this.processPayPal(paymentMethod.email, order.total);
      case 'bank_transfer':
        return this.processBankTransfer({<CURSOR>});
      case 'crypto':
        return this.processCrypto(
          paymentMethod.walletAddress,
          paymentMethod.currency,
          order.total
        );
      default:
        throw new Error('Unknown payment method');
    }
  }

  private async processCreditCard(cardNumber: string, amount: number): Promise<boolean> {
    console.log(`Processing credit card payment: ${amount}`);
    return true;
  }

  private async processPayPal(email: string, amount: number): Promise<boolean> {
    console.log(`Processing PayPal payment: ${amount}`);
    return true;
  }

  private async processBankTransfer(details: {
    accountNumber: string;
    routingNumber: string;
    amount: number;
  }): Promise<boolean> {
    console.log(`Processing bank transfer: ${details.amount}`);
    return true;
  }

  private async processCrypto(
    walletAddress: string,
    currency: string,
    amount: number
  ): Promise<boolean> {
    console.log(`Processing crypto payment: ${amount} ${currency}`);
    return true;
  }
}

// Expected: completion should fill in the bank transfer parameters
