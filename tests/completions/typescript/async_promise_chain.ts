interface ApiClient {
  get<T>(url: string): Promise<T>;
  post<T>(url: string, data: unknown): Promise<T>;
}

interface User {
  id: string;
  name: string;
  email: string;
}

interface Post {
  id: string;
  userId: string;
  title: string;
  content: string;
  createdAt: Date;
}

interface Comment {
  id: string;
  postId: string;
  userId: string;
  text: string;
}

class BlogService {
  constructor(private client: ApiClient) {}

  async getUserPosts(userId: string): Promise<Post[]> {
    try {
      const user = await this.client.get<User>(`/users/${userId}`);
      if (!user) {
        throw new Error('User not found');
      }

      const posts = await this.client.get<Post[]>(`/users/${userId}/posts`);
      return posts;
    } catch (error) {
      console.error('Error fetching user posts:', error);
      throw error;
    }
  }

  async getPostWithComments(postId: string): Promise<{
    post: Post;
    comments: Comment[];
    author: User;
  }> {
    const [post, comments] = await Promise.all([<CURSOR>]);

    const author = await this.client.get<User>(`/users/${post.userId}`);

    return {
      post,
      comments,
      author,
    };
  }

  async createPost(userId: string, title: string, content: string): Promise<Post> {
    const postData = {
      userId,
      title,
      content,
      createdAt: new Date(),
    };

    return await this.client.post<Post>('/posts', postData);
  }
}

// Expected: completion should add the Promise.all array elements
