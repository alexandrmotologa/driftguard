# Mathematical Foundations of Trajectory Divergence

driftguard uses non-parametric statistical metrics and Markov chain transition modeling to detect silent reasoning drift in production language model agents.

## 1. Step Count Distribution: 1D Wasserstein Distance

Traditional mean and variance checks fail to detect bimodal shifts or subtle tail expansions in agent execution lengths. driftguard computes the first Wasserstein distance (also known as Earth Mover's Distance) between the baseline step count distribution $P$ and the empirical rolling distribution $Q$.

### Formulation

For two probability distributions $P$ and $Q$ on the real line with cumulative distribution functions $F_P(x)$ and $F_Q(x)$, the 1D Wasserstein distance is defined as:

$$W_1(P, Q) = \int_{-\infty}^{\infty} |F_P(x) - F_Q(x)| \, dx$$

For discrete empirical samples $X = \{x_1, \dots, x_n\}$ from $P$ and $Y = \{y_1, \dots, y_m\}$ from $Q$:

1. Sort both sample sets in ascending order.
2. Evaluate the cumulative distribution step functions:
   $$F_P(t) = \frac{1}{n} \sum_{i=1}^n \mathbf{1}_{x_i \le t}, \quad F_Q(t) = \frac{1}{m} \sum_{j=1}^m \mathbf{1}_{y_j \le t}$$
3. Integrate the absolute difference across all partition intervals defined by sorted union points $\{u_k\} = \text{sort}(X \cup Y)$:
   $$W_1(P, Q) = \sum_{k=1}^{|U|-1} (u_{k+1} - u_k) \cdot |F_P(u_k) - F_Q(u_k)|$$

### Decision Threshold

An anomaly is flagged when:
$$W_1(P, Q) > \theta_{\text{step}}$$

Typical production thresholds set $\theta_{\text{step}} \in [1.5, 2.5]$, corresponding to a shift of more than 1.5 to 2.5 full reasoning steps across the window.

---

## 2. Markov Chain Tool Transition Modeling

Autonomous agents execute multi-step tool calls that form a first-order Markov chain. We define a discrete state space:

$$S = \{T_{\text{start}}, t_1, t_2, \dots, t_K, T_{\text{end}}\}$$

Where $t_k$ represents a distinct tool call, $T_{\text{start}}$ is the initial prompt state, and $T_{\text{end}}$ is the terminal response state.

### Empirical Transition Probability

For a window of observed traces, let $C(t_i, t_j)$ be the count of consecutive transitions from tool $t_i$ to tool $t_j$. The transition probability is:

$$P(t_j \mid t_i) = \frac{C(t_i, t_j) + \alpha}{\sum_{k} (C(t_i, t_k) + \alpha)}$$

Where $\alpha \ge 0$ is a Laplace smoothing parameter (default $\alpha = 0.01$).

### Transition Matrix Divergence Metric

Let $M_{\text{base}}$ be the Golden Baseline transition matrix and $M_{\text{curr}}$ be the rolling window transition matrix. The overall matrix perturbation metric $\Delta M$ is computed as the normalized entrywise absolute deviation:

$$\Delta M = \frac{1}{|S|^2} \sum_{i=1}^{|S|} \sum_{j=1}^{|S|} |M_{\text{curr}}(i, j) - M_{\text{base}}(i, j)|$$

Additionally, driftguard computes the maximum single edge perturbation:

$$\Delta M_{\max} = \max_{i, j} |M_{\text{curr}}(i, j) - M_{\text{base}}(i, j)|$$

### Novel Transition Detection

If an observed edge $(t_i, t_j)$ has baseline probability $M_{\text{base}}(i, j) = 0$ and occurs repeatedly in the rolling window ($C_{\text{curr}}(i, j) \ge k_{\text{threshold}}$), driftguard raises a **Novel Transition Anomaly**. This often signals hallucinated arguments or unexpected branching caused by model checkpoint updates.

---

## 3. Tool Loop and Recursion Detection

Language model degradation frequently manifests as self-loops or cyclic tool invocations.

### Self-Loop Intensity

For each tool $t_i$, the self-loop probability is:

$$L(t_i) = P(t_i \mid t_i)$$

If $L(t_i) > \tau_{\text{loop}}$ (default 0.25) where baseline $L_{\text{base}}(t_i) \approx 0$, a **Tool Recursion Warning** is triggered.

### Cycle Detection

Traces are inspected for cyclic sub-sequences $t_a \to t_b \to t_a \to t_b$ with length $\ge 3$. Traces containing redundant cycles are isolated and preserved in the incident store for SRE inspection.

---

## 4. Deterministic Guardrail Verification

Certain tool orderings are strictly required for security and financial integrity (such as running authorization before executing payments).

Let $G = (V, E_{\text{forbidden}})$ represent forbidden directed transitions. If any individual trace contains a transition $(t_u, t_v) \in E_{\text{forbidden}}$, driftguard instantly triggers a critical alert regardless of statistical sample sizes.
