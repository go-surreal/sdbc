package sdbc

//func TestClientSubscribeContextCanceled(t *testing.T) {
//	t.Parallel()
//
//	done := make(chan struct{})
//
//	go func() {
//		defer close(done)
//
//		ctx := context.Background()
//
//		client, cleanup := prepareClient(ctx, t)
//		cleanup()
//
//		ctx, cancel := context.WithCancel(ctx)
//		cancel()
//
//		client.connCtx = ctx
//
//		client.subscribe()
//	}()
//
//	select {
//	case <-done:
//	case <-time.After(5 * time.Second):
//		t.Fatal("test timed out")
//	}
//}
//
//func TestClientSubscribeContextDeadlineExceeded(t *testing.T) {
//	t.Parallel()
//
//	done := make(chan struct{})
//
//	go func() {
//		defer close(done)
//
//		ctx := context.Background()
//
//		client, cleanup := prepareClient(ctx, t)
//		cleanup()
//
//		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second))
//		defer cancel()
//
//		client.connCtx = ctx
//
//		client.subscribe()
//	}()
//
//	select {
//	case <-done:
//	case <-time.After(5 * time.Second):
//		t.Fatal("test timed out")
//	}
//}
