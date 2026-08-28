import { Component, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  render() {
    if (this.state.error) {
      return (
        <div className="p-6">
          <h2 className="text-red-400 font-semibold mb-2">页面渲染出错</h2>
          <pre className="text-xs text-obsidian-400 whitespace-pre-wrap">
            {this.state.error.message}
          </pre>
          <pre className="text-xs text-obsidian-600 whitespace-pre-wrap mt-2">
            {this.state.error.stack}
          </pre>
          <button
            onClick={() => this.setState({ error: null })}
            className="mt-4 px-4 py-2 bg-amber-500 text-obsidian-950 rounded-lg text-sm font-semibold"
          >
            重试
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
