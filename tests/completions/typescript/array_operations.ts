interface Transaction {
  id: string;
  amount: number;
  type: 'debit' | 'credit';
  timestamp: Date;
  category: string;
  description: string;
}

interface Account {
  id: string;
  balance: number;
  transactions: Transaction[];
}

class AccountAnalyzer {
  calculateBalance(account: Account): number {
    return account.transactions.reduce((balance, transaction) => {
      if (transaction.type === 'credit') {
        return balance + transaction.amount;
      } else {
        return balance - transaction.amount;
      }
    }, 0);
  }

  getTransactionsByCategory(account: Account, category: string): Transaction[] {
    return account.transactions.filter(t => <CURSOR>);
  }

  getMonthlySummary(account: Account, year: number, month: number): {
    credits: number;
    debits: number;
    net: number;
  } {
    const monthTransactions = account.transactions.filter(t => {
      const txDate = new Date(t.timestamp);
      return txDate.getFullYear() === year && txDate.getMonth() === month;
    });

    const credits = monthTransactions
      .filter(t => t.type === 'credit')
      .reduce((sum, t) => sum + t.amount, 0);

    const debits = monthTransactions
      .filter(t => t.type === 'debit')
      .reduce((sum, t) => sum + t.amount, 0);

    return {
      credits,
      debits,
      net: credits - debits,
    };
  }
}

// Expected: completion should filter transactions by category
