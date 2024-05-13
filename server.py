pre_n = 1
node = [4,10,16,22,28,34,40,46]
for f in node:
    #print(f"f = {f}, C = N/4: {(3*f+1)*4}, C = N/3: {(3*f+1)*3}, C = N/2: {(3*f+1)*2}, ")
    n = f/3
    print(f"Node Number:{f}, 相比上一次增加{n*n/(pre_n*pre_n)}")
    pre_n = n