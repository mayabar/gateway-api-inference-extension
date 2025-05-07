import streamlit as st
import requests
import subprocess
import time
st.set_page_config(layout="wide")
# Define system prompt value separately

# ---- Auto-detect IP from Kubernetes Gateway ----
def get_gateway_ip():
    try:
        result = subprocess.run(
            ["kubectl", "get", "gateway/inference-gateway", "-o", "jsonpath={.status.addresses[0].value}"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError:
        return "localhost"

# ---- Sidebar: Shared Config ----
st.sidebar.title("🧠 Model Config")

default_system_prompt="You're the best chatbot. Be helpful and smart."
ip = st.sidebar.text_input("Router IP", "localhost")
port = st.sidebar.text_input("Router Port", "8080")

model = st.sidebar.selectbox("Model", [
    "meta-llama/Llama-3.1-8B-Instruct",
    "mistralai/Mistral-7B-Instruct-v0.2",
    "Qwen/Qwen2.5-1.5B-Instruct",

])
max_tokens_str = st.sidebar.text_input("Max Tokens", value="100")
try:
    max_tokens = int(max_tokens_str)
except ValueError:
    st.sidebar.error("Max Tokens must be a number")
    max_tokens = 30  # fallback default

if st.sidebar.button("🗑️ Clear All Chats"):
    st.session_state.clear()

# ---- Main Title ----
st.title("🤖 Disaggregated Prefill-Decode Chatbot")

# ---- Initialize Chat Session States ----
for i in range(1, 4):
    key = f"chat{i}_history"
    if key not in st.session_state:
        st.session_state[key] = []

# ---- Chatbot Panels ----
cols = st.columns([1, 1])
for idx, col in enumerate(cols):
    chat_id = f"chat{idx+1}"
    history_key = f"{chat_id}_history"
    session_id = f"{chat_id}-session"

    with col:
        st.subheader(f"💬 {chat_id.capitalize()}")
        # Default prompt for all
        if chat_id == "chat1":
            default_user_prompt = "Whats the weather like in Tokyo tomorrow?"
        if chat_id == "chat2":
            default_user_prompt = "Write a short story about a young inventor who creates a robot to help with daily chores, but things dont go as planned.\n "
        
        # System prompt input
        system_prompt = st.text_area(
            f"System Prompt ({chat_id})",
            value=default_system_prompt,
            key=f"{chat_id}_system_prompt"
        )

        # User prompt input
        prompt = st.text_area(
            f"User Prompt ({chat_id})",
            value=default_user_prompt,
            key=f"{chat_id}_input"
        )

        # Send button
        if st.button(f"Send in {chat_id}", key=f"{chat_id}_send") and prompt:
            url = f"http://{ip}:{port}/v1/completions"
            final_prompt = f"{system_prompt.strip()}\n\n{prompt.strip()}"
            payload = {
                "model": model,
                "prompt": final_prompt,
                "max_tokens": max_tokens,
                "temperature": 0,
                "session_id": session_id
            }

            with st.spinner("🧠 Thinking..."):
                start_time = time.time()
                try:
                    res = requests.post(url, json=payload, timeout=60)
                    duration = time.time() - start_time
                    data = res.json() if res.status_code == 200 else {"error": res.text}
                except Exception as e:
                    duration = time.time() - start_time
                    data = {"error": str(e)}

            st.session_state[history_key].append((prompt, data, duration))

        # Show chat history
        for i, entry in enumerate(st.session_state[history_key][::-1], 1):
            if len(entry) == 3:
                _, res, duration = entry
                # st.markdown(f"🕒 Took **{duration:.2f} seconds**")
            else:
                _, res = entry

            if "choices" in res:
                st.success(res["choices"][0]["text"])
            else:
                st.error(res.get("error", "Unknown error"))
