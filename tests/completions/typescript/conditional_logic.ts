interface Product {
  id: string;
  name: string;
  price: number;
  category: string;
  inStock: boolean;
  discount?: number;
}

interface Cart {
  items: Array<{ product: Product; quantity: number }>;
  userId: string;
}

class PricingService {
  calculateItemPrice(product: Product, quantity: number): number {
    let basePrice = product.price * quantity;

    if (product.discount) {
      basePrice = basePrice * (1 - product.discount);
    }

    return basePrice;
  }

  calculateTotal(cart: Cart): number {
    let total = 0;

    for (const item of cart.items) {
      if (<CURSOR>) {
        continue;
      }
      total += this.calculateItemPrice(item.product, item.quantity);
    }

    return total;
  }

  applyShippingDiscount(total: number, cart: Cart): number {
    if (cart.items.length > 5) {
      return total * 0.9;
    }
    return total;
  }
}

// Expected: completion should add a condition to skip out-of-stock items
